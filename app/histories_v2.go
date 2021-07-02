package app

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/models"
)

func HistoriesV2Handler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		messages := make([]*models.MongoMessage, 0)
		return c.JSON(http.StatusOK, messages)
	}
}
