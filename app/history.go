package app

import (
	"net/http"

	"github.com/topfreegames/mqtt-history/mongoclient"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/logger"
)

// HistoryHandler is the handler responsible for sending the rooms history to the player
func HistoryHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "History")
		topic := c.ParamValues()[0]
		userID, from, limit, _ := ParseHistoryQueryParams(c, app.Defaults.LimitOfMessages)
		authenticated, _, err := IsAuthorized(c.StdContext(), app, userID, topic)
		if err != nil {
			return err
		}

		logger.Logger.Debugf(
			"user %s (authenticated=%v) is asking for history for topic %s with args from=%d and limit=%d",
			userID, authenticated, topic, from, limit)

		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		collection := app.Defaults.MongoMessagesCollection
		messages := mongoclient.GetMessages(
			c,
			mongoclient.QueryParameters{
				Topic:      topic,
				From:       from,
				Limit:      limit,
				Collection: collection,
			},
		)
		return c.JSON(http.StatusOK, messages)

	}
}
