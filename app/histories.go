package app

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/logger"
	"github.com/topfreegames/mqtt-history/models"
)

// HistoriesHandler is the handler responsible for sending multiples rooms history to the player
func HistoriesHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "Histories")
		topicPrefix := c.ParamValues()[0]
		userID := c.QueryParam("userid")
		topicsSuffix := strings.Split(c.QueryParam("topics"), ",")
		topics := make([]string, len(topicsSuffix))
		from, err := strconv.ParseInt(c.QueryParam("from"), 10, 64)
		limit, err := strconv.ParseInt(c.QueryParam("limit"), 10, 64)
		for i, topicSuffix := range topicsSuffix {
			topics[i] = topicPrefix + "/" + topicSuffix
		}
		if limit == 0 {
			limit = app.Defaults.LimitOfMessages
		}

		if from == 0 {
			from = time.Now().Unix()
		}

		logger.Logger.Debugf("user %s is asking for histories for topicPrefix %s with args topics=%s from=%d and limit=%d", userID, topicPrefix, topics, from, limit)
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
				topicMessages := SelectFromCollection(c, topic, from, limit, collection)
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

