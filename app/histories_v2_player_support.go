package app

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/topfreegames/mqtt-history/logger"
	"github.com/topfreegames/mqtt-history/mongoclient"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/models"
)

func HistoriesV2PSHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "HistoriesV2PlayerSupport")
		topic := c.ParamValues()[0]

		userID, _, limit, isBlocked := ParseHistoryQueryParams(c, app.Defaults.LimitOfMessages)
		authenticated, _, err := IsAuthorized(c.StdContext(), app, userID, topic)
		if err != nil {
			return err
		}

		initialDateParamsFilter := c.QueryParam("initialDate")
		finalDateParamsFilter := c.QueryParam("finalDate")
		from := transformDate(initialDateParamsFilter)
		to := transformDate(finalDateParamsFilter)

		logger.Logger.Debugf(
			"user %s (authenticated=%v) is asking for history v2 for topic %s with args from=%d to=%d and limit=%d",
			userID, authenticated, topic, from, to, limit)

		if !authenticated {
			return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
		}

		messages := make([]*models.MessageV2, 0)
		collection := app.Defaults.MongoMessagesCollection
		messages = mongoclient.GetMessagesPlayerSupportV2WithParameter(c, topic, from, to, limit, collection, isBlocked)

		return c.JSON(http.StatusOK, messages)
	}
}

func transformDate(dateParamsFilter string) int64 {

	res := strings.Split(dateParamsFilter, "/")
	day, _ := strconv.Atoi(res[0])
	month, _ := strconv.Atoi(res[1])
	year, _ := strconv.Atoi(res[2])

	return time.Date(year, time.Month(month), day, 23, 59, 59, 59, time.UTC).Unix()
}
