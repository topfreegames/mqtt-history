package app

import (
	"net/http"
	"sync"

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

		var wg sync.WaitGroup
		var mu sync.Mutex
		// guarantees ordering in responses payload
		topicsMessagesMap := make(map[string][]*models.MessageV2, len(authorizedTopics))
		for _, topic := range authorizedTopics {
			wg.Add(1)
			go func(topicsMessagesMap map[string][]*models.MessageV2, topic string) {
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
			}(topicsMessagesMap, topic)
		}
		wg.Wait()
		// guarantees ordering in responses payload
		for _, topic := range authorizedTopics {
			messages = append(messages, topicsMessagesMap[topic]...)
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
