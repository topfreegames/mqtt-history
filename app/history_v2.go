package app

import (
	"net/http"

	"github.com/topfreegames/mqtt-history/mongoclient"

	"github.com/topfreegames/mqtt-history/logger"

	"github.com/topfreegames/mqtt-history/models"

	"github.com/labstack/echo"
)

// HistoryHandler is the handler responsible for sending the rooms history to the player
func HistoryV2Handler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "HistoryV2")
		topic := c.ParamValues()[0]
		userID, from, limit, isBlocked := ParseHistoryQueryParams(c, app.Defaults.LimitOfMessages)
		authenticated, _, err := IsAuthorized(c.StdContext(), app, userID, topic)
		if err != nil {
			return err
		}

		logger.Logger.Debugf(
			"user %s (authenticated=%v) is asking for history v2 for topic %s with args from=%d and limit=%d",
			userID, authenticated, topic, from, limit)

		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		messages := make([]*models.MessageV2, 0)
		collection := app.Defaults.MongoMessagesCollection
		messages = mongoclient.GetMessagesV2WithParameter(
			c,
			mongoclient.QueryParameters{
				Topic:      topic,
				From:       from,
				Limit:      limit,
				Collection: collection,
				IsBlocked:  isBlocked,
			},
		)

		if len(messages) > 0 {
			gameId := messages[0].GameId
			if metricTagsMap, ok := c.Get("metricTagsMap").(map[string]interface{}); ok {
				metricTagsMap["gameID"] = gameId
			}
		}

		return c.JSON(http.StatusOK, messages)
	}
}
