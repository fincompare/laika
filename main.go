package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/MEDIGO/laika/api"
	"github.com/MEDIGO/laika/notifier"
	"github.com/MEDIGO/laika/store"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/urfave/cli"
	graceful "gopkg.in/tylerb/graceful.v1"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	app := cli.NewApp()
	app.Author = "MEDIGO GmbH"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "port",
			Value:  "8000",
			Usage:  "Service port",
			EnvVar: "LAIKA_PORT",
		},
		cli.IntFlag{
			Name:   "timeout",
			Value:  10,
			Usage:  "Shutdown timeout",
			EnvVar: "LAIKA_TIMEOUT",
		},
		cli.StringFlag{
			Name:   "mysql-host",
			Value:  "db",
			Usage:  "MySQL host",
			EnvVar: "LAIKA_MYSQL_HOST",
		},
		cli.StringFlag{
			Name:   "mysql-port",
			Value:  "3306",
			Usage:  "MySQL port",
			EnvVar: "LAIKA_MYSQL_PORT",
		},
		cli.StringFlag{
			Name:   "mysql-username",
			Value:  "root",
			Usage:  "MySQL username",
			EnvVar: "LAIKA_MYSQL_USERNAME",
		},
		cli.StringFlag{
			Name:   "mysql-password",
			Value:  "root",
			Usage:  "MySQL password",
			EnvVar: "LAIKA_MYSQL_PASSWORD",
		},
		cli.StringFlag{
			Name:   "mysql-dbname",
			Value:  "laika",
			Usage:  "MySQL dbname",
			EnvVar: "LAIKA_MYSQL_DBNAME",
		},
		cli.StringFlag{
			Name:   "statsd-host",
			Value:  "localhost",
			Usage:  "Statsd host",
			EnvVar: "LAIKA_STATSD_HOST",
		},
		cli.StringFlag{
			Name:   "statsd-port",
			Value:  "8125",
			Usage:  "Statsd port",
			EnvVar: "LAIKA_STATSD_PORT",
		},
		cli.StringFlag{
			Name:   "root-username",
			Usage:  "Root username",
			Value:  "root",
			EnvVar: "LAIKA_ROOT_USERNAME",
		},
		cli.StringFlag{
			Name:   "root-password",
			Usage:  "Root password",
			Value:  "root",
			EnvVar: "LAIKA_ROOT_PASSWORD",
		},
		cli.StringFlag{
			Name:   "slack-webhook-url",
			Usage:  "Slack webhook URL",
			EnvVar: "LAIKA_SLACK_WEBHOOK_URL",
		},
		cli.StringFlag{
			Name:   "aws-secret-id",
			Usage:  "AWS Secret ID",
			EnvVar: "LAIKA_AWS_SECRET_ID",
		},
		cli.StringFlag{
			Name:   "aws-profile",
			Usage:  "AWS Profile",
			EnvVar: "LAIKA_AWS_PROFILE",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "run",
			Usage: "Runs laika's feature flag service",
			Action: func(c *cli.Context) error {
				if secretID := c.GlobalString("aws-secret-id"); secretID != "" {
					awsProfile := c.GlobalString("aws-profile")
					awsSession := session.Must(session.NewSessionWithOptions(session.Options{
						SharedConfigState: session.SharedConfigEnable,
						Profile: awsProfile,
					}))
					ssm := secretsmanager.New(awsSession)
					secretValue, err := ssm.GetSecretValue(&secretsmanager.GetSecretValueInput{
						SecretId: &secretID,
					})
					if err != nil {
						log.Panic("Cannot read aws-secret-id:", err)
					}
					type DbSecretAws struct {
						User string `json:"username"`
						Pass string `json:"password"`
						Host string `json:"host"`
						Port int    `json:"port"`
						Name string `json:"schema"`
					}
					//log.SetLevel(log.DebugLevel)
					var decodedSecrets DbSecretAws
					json.Unmarshal([]byte(*secretValue.SecretString), &decodedSecrets)
					log.Debugln("data-retrieved", decodedSecrets)
					c.GlobalSet("mysql-username", decodedSecrets.User)
					c.GlobalSet("mysql-password", decodedSecrets.Pass)
					c.GlobalSet("mysql-host",     decodedSecrets.Host)
					c.GlobalSet("mysql-port",     fmt.Sprint(decodedSecrets.Port))
					c.GlobalSet("mysql-dbname",   decodedSecrets.Name)
					log.Debugln(c.GlobalFlagNames())
					log.Debugln(c.GlobalString("mysql-username"))
					log.Debugln(c.GlobalString("mysql-password"))
					log.Debugln(c.GlobalString("mysql-host"))
					log.Debugln(c.GlobalString("mysql-port"))
					log.Debugln(c.GlobalString("mysql-dbname"))
				}
				store, err := store.NewMySQLStore(
					c.GlobalString("mysql-username"),
					c.GlobalString("mysql-password"),
					c.GlobalString("mysql-host"),
					c.GlobalString("mysql-port"),
					c.GlobalString("mysql-dbname"),
				)

				if err != nil {
					return fmt.Errorf("failed to create store: %s", err)
				}

				if err := store.Ping(); err != nil {
					return fmt.Errorf("could not ping database: %s", err)
				}

				if err := store.Migrate(); err != nil {
					return fmt.Errorf("failed to migrate store schema: %s", err)
				}

				if _, err := store.State(); err != nil {
					return fmt.Errorf("failed to compute initial state: %s", err)
				}

				stats, err := statsd.New(c.GlobalString("statsd-host") + ":" + c.GlobalString("statsd-port"))
				if err != nil {
					return fmt.Errorf("failed to create Statsd client: %s", err)
				}

				notifier := notifier.NewSlackNotifier(c.GlobalString("slack-webhook-url"))

				server, err := api.NewServer(api.ServerConfig{
					RootUsername: c.GlobalString("root-username"),
					RootPassword: c.GlobalString("root-password"),
					Store:        store,
					Stats:        stats,
					Notifier:     notifier,
				})
				if err != nil {
					return fmt.Errorf("failed to create server: %s", err)

				}

				log.Info("Starting server on port ", c.GlobalString("port"))
				log.Fatal(graceful.RunWithErr(":"+c.GlobalString("port"), time.Duration(c.Int("timeout"))*time.Second, server))
				log.Info("Server exiting")
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
