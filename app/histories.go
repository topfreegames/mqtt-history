package app

import (
	"net/http"

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

		// retrieve messages
		messages := make([]*models.Message, 0)
		if app.Defaults.MongoEnabled {
			collection := app.Defaults.MongoMessagesCollection

			for _, topic := range authorizedTopics {
				topicMessages := mongoclient.GetMessages(c, topic, from, limit, collection)
				messages = append(messages, topicMessages...)
			}
			return c.JSON(http.StatusOK, messages)
		}

		bucketQnt := app.Defaults.BucketQuantityOnSelect
		currentBucket := app.Bucket.Get(from)

		for _, topic := range authorizedTopics {
			topicMessages := selectFromBuckets(c.StdContext(), bucketQnt, int(limit), currentBucket, topic, app.Cassandra)
			messages = append(messages, topicMessages...)
		}

		return c.JSON(http.StatusOK, messages)
	}
}
