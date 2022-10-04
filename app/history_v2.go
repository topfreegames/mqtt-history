package app

import (
	"net/http"

	"github.com/topfreegames/mqtt-history/mongoclient"
	"github.com/uber-go/zap"

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

		app.Logger.Debug(
			"Request received",
			zap.String("route", "HistoryV2"),
			zap.String("user", userID),
			zap.Bool("authenticated", authenticated),
			zap.String("topic", topic),
			zap.Int64("from", from),
			zap.Int64("limit", limit),
		)

		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		messages := make([]*models.MessageV2, 0)
		collection := app.Defaults.MongoMessagesCollection
		messages, err = mongoclient.GetMessagesV2WithParameter(
			c,
			mongoclient.QueryParameters{
				Topic:      topic,
				From:       from,
				Limit:      limit,
				Collection: collection,
				IsBlocked:  isBlocked,
			},
		)

		if err != nil {
			return err
		}

		if len(messages) > 0 {
			gameId := messages[0].GameId
			if metricTagsMap, ok := c.Get("metricTagsMap").(map[string]interface{}); ok {
				metricTagsMap["gameID"] = gameId
			}
		}

		return c.JSON(http.StatusOK, messages)
	}
}
