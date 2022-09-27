package app

import (
	"net/http"
	"time"

	"github.com/topfreegames/mqtt-history/logger"
	"github.com/topfreegames/mqtt-history/mongoclient"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/models"
)

func HistoriesV2PSHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "HistoriesV2PlayerSupport")
		userID, playerId, topic, limit, isBlocked := ParseHistoryPSQueryParams(c, app.Defaults.LimitOfMessages)

		initialDateParamsFilter := c.QueryParam("initialDate")
		from, err := transformDate(initialDateParamsFilter, true)
		if err != nil {
			logger.Logger.Warningf("Error: %s", err.Error())
			return c.JSON(http.StatusUnprocessableEntity, "Error getting initialDate parameter.")
		}

		finalDateParamsFilter := c.QueryParam("finalDate")
		to, err := transformDate(finalDateParamsFilter, false)
		if err != nil {
			logger.Logger.Warningf("Error: %s", err.Error())
			return c.JSON(http.StatusUnprocessableEntity, "Error getting finalDate parameter.")
		}

		logger.Logger.Debugf(
			"user %s is asking for history v2 for topic %s with date args from=%d to=%d and limit=%d",
			userID, from, to, limit)

		messages := make([]*models.MessageV2, 0)
		collection := app.Defaults.MongoMessagesCollection
		messages = mongoclient.GetMessagesPlayerSupportV2WithParameter(c, topic, from, to, limit, collection, isBlocked, playerId)

		gameId := messages[0].GameId
		metricTagMap := c.Get("metricTagsMap").(map[string]interface{})
		if metricTagMap != nil {
			metricTagMap["gameID"] = gameId
			logger.Logger.Debugf("provided gameID: %s", metricTagMap["gameID"])
		}

		return c.JSON(http.StatusOK, messages)
	}
}

func transformDate(dateParamsFilter string, isInitial bool) (int64, error) {
	utcFormat := "2006-01-02"
	t, err := time.Parse(utcFormat, dateParamsFilter)
	if err != nil {
		return 0, err
	}

	if !isInitial {
		t = t.Add(time.Hour*23 + time.Minute*59 + time.Second*59)
	}

	return t.Unix(), err
}
