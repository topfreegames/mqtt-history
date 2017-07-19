// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app

import (
	"fmt"
	"strings"

	"github.com/labstack/echo"
	newrelic "github.com/newrelic/go-agent"
)

//GetTX returns new relic transaction
func GetTX(c echo.Context) newrelic.Transaction {
	tx := c.Get("txn")
	if tx == nil {
		return nil
	}

	return tx.(newrelic.Transaction)
}

//WithSegment adds a segment to new relic transaction
func WithSegment(name string, c echo.Context, f func() error) error {
	tx := GetTX(c)
	if tx == nil {
		return f()
	}
	segment := newrelic.StartSegment(tx, name)
	defer segment.End()
	return f()
}

func authenticate(app *App, userID string, topics ...string) (bool, []interface{}, error) {
	rc := app.RedisClient.Pool.Get()
	defer rc.Close()
	rc.Send("MULTI")
	rc.Send("GET", userID)
	for _, topic := range topics {
		rc.Send("GET", fmt.Sprintf("%s-%s", userID, topic))

		pieces := strings.Split(topic, "/")
		pieces[len(pieces)-1] = "+"
		wildtopic := strings.Join(pieces, "/")
		rc.Send("GET", fmt.Sprintf("%s-%s", userID, wildtopic))
	}
	r, err := rc.Do("EXEC")
	if err != nil {
		return false, nil, err
	}
	authorizedTopics := []interface{}{}
	redisResults := (r.([]interface{}))
	for i, redisResp := range redisResults[1:] {
		if redisResp != nil {
			authorizedTopics = append(authorizedTopics, topics[i/2])
		}
	}

	return redisResults[0] != nil && len(authorizedTopics) > 0, authorizedTopics, nil
}
