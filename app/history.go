package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/logger"
)

// HistoryHandler is the handler responsible for sending the rooms history to the player
func HistoryHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "History")
		topic := c.ParamValues()[0]
		userID := c.QueryParam("userid")
		from, err := strconv.ParseInt(c.QueryParam("from"), 10, 64)
		limit, err := strconv.Atoi(c.QueryParam("limit"))

		if limit == 0 {
			limit = app.Defaults.LimitOfMessages
		}

		if from == 0 {
			from = time.Now().Unix()
		}

		authenticated, _, err := authenticate(c.StdContext(), app, userID, topic)
		if err != nil {
			return err
		}

		logger.Logger.Debugf(
			"user %s is asking for history for topic %s with args from=%d and limit=%d",
			userID, topic, from, limit)
		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		qnt := app.Defaults.BucketQuantityOnSelect
		messages := app.Cassandra.SelectMessagesInBucket(c.StdContext(),
			topic,
			from, qnt, limit)

		return c.JSON(http.StatusOK, messages)
	}
}

// HistorySinceHandler is the handler responsible for sending the rooms history to the player based in a initial date
func HistorySinceHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "HistorySince")
		topic := c.ParamValues()[0]
		userID := c.QueryParam("userid")
		from, err := strconv.ParseInt(c.QueryParam("from"), 10, 64)
		limit, err := strconv.Atoi(c.QueryParam("limit"))
		since, err := strconv.ParseInt(c.QueryParam("since"), 10, 64)

		now := int64(time.Now().Unix())
		if since > now {
			errorString := fmt.Sprintf("user %s is asking for history for topic %s with args from=%d, limit=%d and since=%d. Since is in the future, setting to 0!", userID, topic, from, limit, since)

			logger.Logger.Errorf(errorString)

			tags := map[string]string{
				"source":     "app",
				"type":       "Since is furure",
				"url":        c.Request().URI(),
				"user-agent": c.Request().Header().Get("User-Agent"),
			}

			raven.CaptureError(errors.New(errorString), tags)
			since = 0
			limit = 100
		}

		defaultLimit := 10
		if limitFromEnv := os.Getenv("HISTORYSINCE_LIMIT"); limitFromEnv != "" {
			defaultLimit, err = strconv.Atoi(limitFromEnv)
		}
		if limit == 0 {
			limit = defaultLimit
		}

		if from == 0 {
			from = time.Now().Unix()
		}

		logger.Logger.Debugf("user %s is asking for history for topic %s with args from=%d, limit=%d and since=%d", userID, topic, from, limit, since)
		authenticated, _, err := authenticate(c.StdContext(), app, userID, topic)
		if err != nil {
			return err
		}

		if !authenticated {
			logger.Logger.Debugf(
				"responded to user %s history for topic %s with args from=%d limit=%d and since=%d with code=%d and message=%s",
				userID, topic, from, limit, since, echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message,
			)
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		messages := app.Cassandra.SelectMessagesBeforeTime(c.StdContext(), topic, from, since, limit)

		var resStr []byte
		resStr, err = json.Marshal(messages)
		if err != nil {
			return err
		}
		logger.Logger.Debugf(
			"responded to user %s history for topic %s with args from=%d limit=%d and since=%d with code=%d and message=%s",
			userID, topic, from, limit, since, http.StatusOK, string(resStr),
		)

		return c.JSON(http.StatusOK, messages)
	}
}
