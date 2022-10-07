package app

import (
	"net/http"
	"sync"

	"github.com/topfreegames/mqtt-history/mongoclient"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/models"
)

// HistoriesHandler is the handler responsible for sending multiples rooms history to the player
func HistoriesHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "Histories")
		topicPrefix := c.ParamValues()[0]
		topicsSuffix, userID, from, limit := ParseHistoriesQueryParams(c, app.Defaults.LimitOfMessages)
		topics := make([]string, len(topicsSuffix))

		for i, topicSuffix := range topicsSuffix {
			topics[i] = topicPrefix + "/" + topicSuffix
		}

		authenticated, authorizedTopics, err := IsAuthorized(c.StdContext(), app, userID, topics...)
		if err != nil {
			return err
		}

		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		messages := make([]*models.Message, 0)
		collection := app.Defaults.MongoMessagesCollection
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
			topicMessages := make([]*models.Message, len(topicsMessagesMap[topic]))
			for idx, topicMessageV2 := range topicsMessagesMap[topic] {
				topicMessages[idx] = mongoclient.ConvertMessageV2ToMessage(topicMessageV2)
			}
			messages = append(messages, topicMessages...)
			if gameID != "" && len(topicsMessagesMap[topic]) > 0 {
				gameID = topicsMessagesMap[topic][0].GameId
			}
		}
		return c.JSON(http.StatusOK, messages)

	}
}
