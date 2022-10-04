package app

import (
	"net/http"

	"github.com/topfreegames/mqtt-history/mongoclient"
	"github.com/uber-go/zap"

	"github.com/labstack/echo"
)

// HistoryHandler is the handler responsible for sending the rooms history to the player
func HistoryHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "History")
		topic := c.ParamValues()[0]
		userID, from, limit, _ := ParseHistoryQueryParams(c, app.Defaults.LimitOfMessages)
		authenticated, _, err := IsAuthorized(c.StdContext(), app, userID, topic)
		if err != nil {
			return err
		}

		app.Logger.Debug(
			"Request received",
			zap.String("route", "History"),
			zap.String("user", userID),
			zap.Bool("authenticated", authenticated),
			zap.String("topic", topic),
			zap.Int64("from", from),
			zap.Int64("limit", limit),
		)

		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		if app.Defaults.MongoEnabled {
			collection := app.Defaults.MongoMessagesCollection
			messages, err := mongoclient.GetMessages(
				c,
				mongoclient.QueryParameters{
					Topic:      topic,
					From:       from,
					Limit:      limit,
					Collection: collection,
				},
			)
			if err != nil {
				return err
			}
			return c.JSON(http.StatusOK, messages)
		}

		bucketQnt := app.Defaults.BucketQuantityOnSelect
		currentBucket := app.Bucket.Get(from)

		messages, err := selectFromBuckets(c.StdContext(),
			bucketQnt, int(limit), currentBucket,
			topic,
			app.Cassandra)

		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, messages)
	}
}
