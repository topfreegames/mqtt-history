package app

import (
	"net/http"

	"github.com/topfreegames/mqtt-history/models"
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
		authenticated, _, err := IsAuthorized(c, app, userID, topic)
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
		messages := make([]*models.Message, 0)
		messagesV2 := mongoclient.GetMessagesV2(
			c,
			mongoclient.QueryParameters{
				Topic:      topic,
				From:       from,
				Limit:      limit,
				Collection: collection,
			},
		)

		var gameID string

		message := make([]*models.Message, len(messagesV2))
		for idx, messageV2 := range messagesV2 {
			message[idx] = mongoclient.ConvertMessageV2ToMessage(messageV2)
		}
		messages = append(messages, message...)

		if len(messages) > 0 {
			gameID = messagesV2[0].GameId
			if metricTagsMap, ok := c.Get("metricTagsMap").(map[string]interface{}); ok {
				metricTagsMap["gameID"] = gameID
			}
		}

		return c.JSON(http.StatusOK, messages)

	}
}
