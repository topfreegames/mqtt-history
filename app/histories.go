package app

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/es"
	"github.com/topfreegames/mqtt-history/logger"
	"gopkg.in/olivere/elastic.v5"
)

// HistoriesHandler is the handler responsible for sending multiples rooms history to the player
func HistoriesHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		esclient := es.GetESClient()
		c.Set("route", "Histories")
		topicPrefix := c.ParamValues()[0]
		userID := c.QueryParam("userid")
		topicsSuffix := strings.Split(c.QueryParam("topics"), ",")
		topics := make([]string, len(topicsSuffix))
		from, err := strconv.Atoi(c.QueryParam("from"))
		limit, err := strconv.Atoi(c.QueryParam("limit"))
		for i, topicSuffix := range topicsSuffix {
			topics[i] = topicPrefix + "/" + topicSuffix
		}
		if limit == 0 {
			limit = 10
		}

		logger.Logger.Debugf("user %s is asking for histories for topicPrefix %s with args topics=%s from=%d and limit=%d", userID, topicPrefix, topics, from, limit)
		authenticated, authorizedTopics, err := authenticate(app, userID, topics...)
		if err != nil {
			return err
		}

		fmt.Println("henrod", authorizedTopics)

		if authenticated {
			boolQuery := elastic.NewBoolQuery()
			topicBoolQuery := elastic.NewBoolQuery()
			topicBoolQuery.Should(elastic.NewTermsQuery("topic", authorizedTopics...))
			boolQuery.Must(topicBoolQuery)

			var searchResults *elastic.SearchResult
			err = WithSegment("elasticsearch", c, func() error {
				searchResults, err = esclient.Search().Index("chat-*").Query(boolQuery).
					Sort("timestamp", false).From(from).Size(limit).Do(context.TODO())
				return err
			})

			if err != nil {
				return err
			}
			messages := []Message{}
			var ttyp Message
			for _, item := range searchResults.Each(reflect.TypeOf(ttyp)) {
				if t, ok := item.(Message); ok {
					messages = append(messages, t)
				}
			}
			return c.JSON(http.StatusOK, messages)
		}
		return c.String(echo.ErrUnauthorized.Code, echo.ErrUnauthorized.Message)
	}
}
