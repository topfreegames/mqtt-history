// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app

import (
	"github.com/labstack/echo"
	newrelic "github.com/newrelic/go-agent"
	"github.com/spf13/viper"
	"github.com/topfreegames/mqtt-history/mongoclient"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Acl struct {
	Id       bson.ObjectId "_id,omitempty"
	Username string        "username"
	Pubsub   []string      "pubsub"
}

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

func MongoSearch(q interface{}) ([]Acl, error) {
	searchResults := []Acl{}
	query := func(c *mgo.Collection) error {
		fn := c.Find(q).All(&searchResults)
		return fn
	}
	search := func() error {
		return mongoclient.GetCollection("mqtt", "mqtt_acl", query)
	}
	err := search()
	return searchResults, err
}

func GetTopics(username string, _topics []string) ([]string, error) {
	if viper.GetBool("mongo.allow_anonymous") {
		return _topics, nil
	}
	var topics []string
	searchResults, err := MongoSearch(bson.M{"username": username, "pubsub": bson.M{"$in": _topics}})
	for _, elem := range searchResults {
		topics = append(topics, elem.Pubsub[0])
	}
	return topics, err
}

func authenticate(app *App, userID string, topics ...string) (bool, []interface{}, error) {
	var allowedTopics, err = GetTopics(userID, topics)
	if err != nil {
		return false, nil, err
	}
	allowed := make(map[string]bool)
	for _, topic := range allowedTopics {
		allowed[topic] = true
	}
	authorizedTopics := []interface{}{}
	for _, topic := range topics {
		if allowed[topic] {
			authorizedTopics = append(authorizedTopics, topic)
		}
	}
	return len(authorizedTopics) > 0, authorizedTopics, nil
}

//func authenticate(app *App, userID string, topics ...string) (bool, []interface{}, error) {
//	rc := app.RedisClient.Pool.Get()
//	defer rc.Close()
//	rc.Send("MULTI")
//	rc.Send("GET", userID)
//	for _, topic := range topics {
//		rc.Send("GET", fmt.Sprintf("%s-%s", userID, topic))
//
//		pieces := strings.Split(topic, "/")
//		pieces[len(pieces)-1] = "+"
//		wildtopic := strings.Join(pieces, "/")
//		rc.Send("GET", fmt.Sprintf("%s-%s", userID, wildtopic))
//	}
//	r, err := rc.Do("EXEC")
//	if err != nil {
//		return false, nil, err
//	}
//	authorizedTopics := []interface{}{}
//	redisResults := (r.([]interface{}))
//	for i, redisResp := range redisResults[1:] {
//		if redisResp != nil {
//			authorizedTopics = append(authorizedTopics, topics[i/2])
//		}
//	}
//
//	return redisResults[0] != nil && len(authorizedTopics) > 0, authorizedTopics, nil
//}
