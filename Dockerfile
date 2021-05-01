# syntax=docker/dockerfile:experimental

ARG APP_API_PATH=/go/src/github.com/fincompare/laika
ARG APP_DASH_PATH=/usr/src/app

FROM node:12.9.1-alpine AS dash
ARG APP_DASH_PATH
WORKDIR ${APP_DASH_PATH}

COPY \
	./dashboard/yarn.lock \
	./dashboard/package.json \
	./
RUN --mount=type=cache,rw,target=./node_modules,source=./dashboard/node_modules \
	yarn install

COPY ./dashboard/ ./
RUN --mount=type=cache,rw,target=./node_modules,source=./dashboard/node_modules \
	yarn build


FROM golang:1.13-alpine AS api
RUN apk --no-cache add gcc g++ make ca-certificates

ARG APP_API_PATH
WORKDIR ${APP_API_PATH}

COPY \
	./go.mod \
	./go.sum \
	./
RUN go mod download

COPY ./ ./
ARG CGO_ENABLED=0
RUN --mount=type=cache,rw,target=./vendor,source=./vendor \
	go build -o bin/laika .

FROM alpine:latest AS runtime

RUN apk add --update --no-cache ca-certificates
RUN update-ca-certificates

ARG APP_API_PATH
ARG APP_DASH_PATH

WORKDIR /home
COPY --from=api  ${APP_API_PATH}/bin/laika ./
COPY --from=dash ${APP_DASH_PATH}/public ./dashboard/public

ENTRYPOINT [ "./laika" ]
CMD [ "run" ]

EXPOSE 8000/tcp

ENV LAIKA_PORT 8000
ENV LAIKA_TIMEOUT 10
ENV LAIKA_MYSQL_HOST db
ENV LAIKA_MYSQL_PORT 3306
ENV LAIKA_MYSQL_USERNAME root
ENV LAIKA_MYSQL_PASSWORD root
ENV LAIKA_MYSQL_DBNAME laika
ENV LAIKA_STATSD_HOST localhost
ENV LAIKA_STATSD_PORT 8125
ENV LAIKA_ROOT_USERNAME root
ENV LAIKA_ROOT_PASSWORD root
ENV LAIKA_SLACK_WEBHOOK_URL ""
ENV LAIKA_AWS_SECRET_ID ""
