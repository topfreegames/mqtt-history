package app

import (
	"net/http"

	"github.com/topfreegames/mqtt-history/models"

	"github.com/labstack/echo"
)

// HistoryHandler is the handler responsible for sending the rooms history to the player
func HistoryV2Handler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		messages := make([]*models.MongoMessage, 0)
		return c.JSON(http.StatusOK, messages)
	}
}
