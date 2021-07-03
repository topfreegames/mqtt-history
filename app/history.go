package app

import (
	"net/http"

	"github.com/topfreegames/mqtt-history/mongoclient"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/logger"
)

// HistoryHandler is the handler responsible for sending the rooms history to the player
func HistoryHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "History")
		topic := c.ParamValues()[0]
		userID, from, limit := ParseHistoryQueryParams(c, app.Defaults.LimitOfMessages)

		authenticated, _, err := IsAuthorized(c.StdContext(), app, userID, topic)
		if err != nil {
			return err
		}

		logger.Logger.Debugf(
			"user %s (authenticated=%v) is asking for history for topic %s with args from=%d and limit=%d",
			userID, authenticated, topic, from, limit)

		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		if app.Defaults.MongoEnabled {
			collection := app.Defaults.MongoMessagesCollection
			messages := mongoclient.GetMessages(c, topic, from, limit, collection)
			return c.JSON(http.StatusOK, messages)
		}

		bucketQnt := app.Defaults.BucketQuantityOnSelect
		currentBucket := app.Bucket.Get(from)

		messages := selectFromBuckets(c.StdContext(),
			bucketQnt, int(limit), currentBucket,
			topic,
			app.Cassandra)

		return c.JSON(http.StatusOK, messages)
	}
}
