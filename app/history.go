package app

import (
	"net/http"
	"sync"

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
		authenticated, authorizedTopics, err := IsAuthorized(c, app, userID, topic)
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

		var wg sync.WaitGroup
		var mu sync.Mutex
		// guarantees ordering in responses payload
		topicsMessagesMap := make(map[string][]*models.MessageV2, len(authorizedTopics))
		for _, topic := range authorizedTopics {
			wg.Add(1)
			go func(topic string) {
				topicMessages := mongoclient.GetMessagesV2(
					c,
					mongoclient.QueryParameters{
						Topic:      topic,
						From:       from,
						Limit:      limit,
						Collection: collection,
					},
				)
				mu.Lock()
				topicsMessagesMap[topic] = topicMessages
				mu.Unlock()
				wg.Done()
			}(topic)
		}
		wg.Wait()
		var gameID string
		// guarantees ordering in responses payload
		for _, topic := range authorizedTopics {
			messages = append(messages)
			if len(topicsMessagesMap[topic]) > 0 {
				gameID = topicsMessagesMap[topic][0].GameId
			}
		}

		if len(messages) > 0 {
			if metricTagsMap, ok := c.Get("metricTagsMap").(map[string]interface{}); ok {
				metricTagsMap["gameID"] = gameID
			}
		}

		return c.JSON(http.StatusOK, messages)

	}
}
