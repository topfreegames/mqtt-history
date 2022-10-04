package app

import (
	"net/http"
	"time"

	"github.com/topfreegames/mqtt-history/mongoclient"
	"github.com/uber-go/zap"

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
			app.Logger.Warn("Error getting initialDate parameter.", zap.Error(err))
			return c.JSON(http.StatusUnprocessableEntity, "Error getting initialDate parameter.")
		}

		finalDateParamsFilter := c.QueryParam("finalDate")
		to, err := transformDate(finalDateParamsFilter, false)
		if err != nil {
			app.Logger.Warn("Error getting finalDate parameter.", zap.Error(err))
			return c.JSON(http.StatusUnprocessableEntity, "Error getting finalDate parameter.")
		}

		app.Logger.Debug(
			"Request received",
			zap.String("route", "HistoriesV2PlayerSupport"),
			zap.String("user", userID),
			zap.String("topic", topic),
			zap.Int64("from", from),
			zap.Int64("to", to),
			zap.Int64("limit", limit),
		)

		messages := make([]*models.MessageV2, 0)
		collection := app.Defaults.MongoMessagesCollection
		messages, err = mongoclient.GetMessagesPlayerSupportV2WithParameter(
			c,
			mongoclient.QueryParameters{
				Topic:      topic,
				From:       from,
				To:         to,
				Limit:      limit,
				Collection: collection,
				IsBlocked:  isBlocked,
				PlayerID:   playerId,
			},
		)

		if err != nil {
			return err
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
