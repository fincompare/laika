package api

import (
	"github.com/labstack/echo"
)

func ListEnvironments(c echo.Context) error {
	return OK(c, getState(c).Environments)
}

/*reateEnvironment(c echo.Context) error {
	eventType := c.Param("type")
	event, err := models.EventForType(eventType)
	if err != nil {
		return Invalid(c, err.Error())
	}

	if err := c.Bind(&event); err != nil {
		return BadRequest(c, "Body must be a valid JSON object")
	}

	state := getState(c)

	valErr, err := event.Validate(state)
	if err != nil {
		return InternalServerError(c, err)
	} else if valErr != nil {
		return BadRequest(c, valErr.Error())
	}

	event, err = event.PrePersist(state)
	if err != nil {
		return InternalServerError(c, err)
	}

	cleanEvent, err := json.Marshal(event)
	if err != nil {
		return InternalServerError(c, err)
	}

	id, err := r.store.Persist(eventType, string(cleanEvent))
	if err != nil {
		return InternalServerError(c, err)
	}

	event.Notify(state, r.notifier)

	return Created(c, struct {
		ID int64 `json:"id"`
	}{ID: id})
}
*/
