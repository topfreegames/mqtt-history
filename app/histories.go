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

		messages := make([]*models.Message, 0)

		collection := app.Defaults.MongoMessagesCollection

		for _, topic := range authorizedTopics {
			topicMessages := mongoclient.GetMessages(
				c,
				mongoclient.QueryParameters{
					Topic:      topic,
					From:       from,
					Limit:      limit,
					Collection: collection,
				},
			)
			messages = append(messages, topicMessages...)
		}
		return c.JSON(http.StatusOK, messages)

	}
}
