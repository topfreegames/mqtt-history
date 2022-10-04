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
		if app.Defaults.MongoEnabled {
			collection := app.Defaults.MongoMessagesCollection
			var wg sync.WaitGroup
			var mu sync.Mutex
			// guarantees ordering in responses payload
			topicsMessagesMap := make(map[string][]*models.Message, len(authorizedTopics))
			for _, topic := range authorizedTopics {
				wg.Add(1)
				go func(topicsMessagesMap map[string][]*models.Message, topic string) {
					topicMessages, er := mongoclient.GetMessages(
						c,
						mongoclient.QueryParameters{
							Topic:      topic,
							From:       from,
							Limit:      limit,
							Collection: collection,
						},
					)
					mu.Lock()
					if er != nil {
						err = er
					} else {
						topicsMessagesMap[topic] = topicMessages
					}
					mu.Unlock()
					wg.Done()
				}(topicsMessagesMap, topic)
			}
			wg.Wait()
			if err != nil {
				return err
			}
			// guarantees ordering in responses payload
			for _, topic := range authorizedTopics {
				messages = append(messages, topicsMessagesMap[topic]...)
			}
			return c.JSON(http.StatusOK, messages)
		}

		bucketQnt := app.Defaults.BucketQuantityOnSelect
		currentBucket := app.Bucket.Get(from)

		for _, topic := range authorizedTopics {
			topicMessages, er := selectFromBuckets(c.StdContext(), bucketQnt, int(limit), currentBucket, topic, app.Cassandra)
			err = er
			messages = append(messages, topicMessages...)
		}

		if err != nil {
			return echo.NewHTTPError(http.StatusBadGateway)
		}

		return c.JSON(http.StatusOK, messages)
	}
}
