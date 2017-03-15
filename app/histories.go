package app

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/es"
	"github.com/topfreegames/mqtt-history/logger"
	"gopkg.in/olivere/elastic.v3"
)

// HistoriesHandler is the handler responsible for sending multiples rooms history to the player
func HistoriesHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		esclient := es.GetESClient()
		c.Set("route", "Histories")
		topicPrefix := c.ParamValues()[0]
		authorizedTopics := []string{}
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
		rc := app.RedisClient.Pool.Get()
		defer rc.Close()
		rc.Send("MULTI")
		rc.Send("GET", userID)
		for _, topic := range topics {
			rc.Send("GET", fmt.Sprintf("%s-%s", userID, topic))
		}
		r, err := rc.Do("EXEC")
		if err != nil {
			return err
		}
		redisResults := (r.([]interface{}))
		for i, redisResp := range redisResults[1:] {
			if redisResp != nil {
				authorizedTopics = append(authorizedTopics, topics[i])
			}
		}

		if redisResults[0] != nil && len(authorizedTopics) > 0 {
			boolQuery := elastic.NewBoolQuery()
			topicBoolQuery := elastic.NewBoolQuery()
			topicBoolQuery.Should(elastic.NewTermsQuery("topic", authorizedTopics))
			boolQuery.Must(topicBoolQuery)

			var searchResults *elastic.SearchResult
			err = WithSegment("elasticsearch", c, func() error {
				searchResults, err = esclient.Search().Index("chat").Query(boolQuery).
					Sort("timestamp", false).From(from).Size(limit).Do()
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
