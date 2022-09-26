package app

import (
	"net/http"

	"github.com/topfreegames/mqtt-history/logger"
	"github.com/topfreegames/mqtt-history/mongoclient"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/models"
)

func HistoriesV2Handler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "HistoriesV2")
		topicPrefix := c.ParamValues()[0]
		topicsSuffix, userID, from, limit := ParseHistoriesQueryParams(c, app.Defaults.LimitOfMessages)
		topics := make([]string, len(topicsSuffix))

		for i, topicSuffix := range topicsSuffix {
			topics[i] = topicPrefix + "/" + topicSuffix
		}

		logger.Logger.Debugf("user %s is asking for histories v2 for topicPrefix %s with args topics=%s from=%d and limit=%d", userID, topicPrefix, topics, from, limit)
		authenticated, authorizedTopics, err := IsAuthorized(c.StdContext(), app, userID, topics...)
		if err != nil {
			return err
		}

		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		// retrieve messages
		messages := make([]*models.MessageV2, 0)
		collection := app.Defaults.MongoMessagesCollection

		for _, topic := range authorizedTopics {
			topicMessages := mongoclient.GetMessagesV2(
				c,
				mongoclient.QueryParameters{
					Topic:      topic,
					From:       from,
					Limit:      limit,
					Collection: collection,
				},
			)
			messages = append(messages, topicMessages...)
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
