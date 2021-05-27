// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
	newrelic "github.com/newrelic/go-agent"
	"github.com/spf13/viper"
	"github.com/topfreegames/extensions/mongo/interfaces"
	"github.com/topfreegames/mqtt-history/mongoclient"
	"gopkg.in/mgo.v2/bson"
)

// ACL is the acl struct
type ACL struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	Username string        `bson:"username"`
	Pubsub   []string      `bson:"pubsub"`
}

type authRequest struct {
	Username string `json:"username"`
	Topic    string `json:"topic"`
}

// GetTX returns new relic transaction
func GetTX(c echo.Context) newrelic.Transaction {
	tx := c.Get("txn")
	if tx == nil {
		return nil
	}

	return tx.(newrelic.Transaction)
}

// WithSegment adds a segment to new relic transaction
func WithSegment(name string, c echo.Context, f func() error) error {
	tx := GetTX(c)
	if tx == nil {
		return f()
	}
	segment := newrelic.StartSegment(tx, name)
	defer segment.End()
	return f()
}

// MongoSearch searches on mongo
func MongoSearch(ctx context.Context, q interface{}) ([]ACL, error) {
	searchResults := make([]ACL, 0)
	query := func(c interfaces.Collection) error {
		fn := c.Find(q).All(&searchResults)
		return fn
	}
	search := func() error {
		return mongoclient.GetCollection(ctx, "mqtt_acl", query)
	}
	err := search()
	return searchResults, err
}

// GetTopics get topics
func GetTopics(ctx context.Context, username string, _topics []string) ([]string, error) {
	if viper.GetBool("mongo.allow_anonymous") {
		return _topics, nil
	}
	var topics []string
	searchResults, err := MongoSearch(ctx, bson.M{"username": username, "pubsub": bson.M{"$in": _topics}})
	if err != nil {
		return nil, err
	}
	for _, elem := range searchResults {
		topics = append(topics, elem.Pubsub[0])
	}
	return topics, err
}

// IsAuthorized returns a boolean indicating whether the user is authorized to read messages
// from at least one of the given topics, and also a slice of all topics on which the user has authorization.
func IsAuthorized(ctx context.Context, app *App, userID string, topics ...string) (bool, []string, error) {
	httpAuthEnabled := app.Config.GetBool("httpAuth.enabled")

	if httpAuthEnabled {
		return httpAuthorize(app, userID, topics)
	}

	return mongoAuthorize(ctx, userID, topics)
}

func httpAuthorize(app *App, userID string, topics []string) (bool, []string, error) {
	timeout := app.Config.GetDuration("httpAuth.timeout") * time.Second
	address := app.Config.GetString("httpAuth.requestURL")

	client := http.Client{
		Timeout: timeout,
	}

	isAuthorized := false
	allowedTopics := make([]string, 0)
	for _, topic := range topics {
		authRequest := authRequest{
			Username: userID,
			Topic:    topic,
		}

		jsonPayload, _ := json.Marshal(authRequest)
		request, _ := http.NewRequest(http.MethodPost, address, bytes.NewReader(jsonPayload))

		credentialsNeeded := app.Config.GetBool("httpAuth.iam.enabled")
		if credentialsNeeded {
			username := app.Config.GetString("httpAuth.iam.credentials.username")
			password := app.Config.GetString("httpAuth.iam.credentials.password")

			request.SetBasicAuth(username, password)
		}

		response, err := client.Do(request)
		// discard response body
		if response != nil && response.Body != nil {
			_, _ = io.Copy(ioutil.Discard, response.Body)
			_ = response.Body.Close()
		}

		if err != nil {
			return false, nil, err
		}

		if response.StatusCode == 200 {
			isAuthorized = true
			allowedTopics = append(allowedTopics, topic)
		}
	}

	return isAuthorized, allowedTopics, nil
}

func mongoAuthorize(ctx context.Context, userID string, topics []string) (bool, []string, error) {
	for _, topic := range topics {
		pieces := strings.Split(topic, "/")
		pieces[len(pieces)-1] = "+"
		wildtopic := strings.Join(pieces, "/")
		topics = append(topics, wildtopic)
	}
	var allowedTopics, err = GetTopics(ctx, userID, topics)
	if err != nil {
		return false, nil, err
	}
	allowed := make(map[string]bool)
	for _, topic := range allowedTopics {
		allowed[topic] = true
	}
	authorizedTopics := make([]string, 0)
	isAuthorized := false
	for _, topic := range topics {
		isAuthorized = isAuthorized || allowed[topic]
		if allowed[topic] && !strings.Contains(topic, "+") {
			authorizedTopics = append(authorizedTopics, topic)
		}
	}
	return isAuthorized, authorizedTopics, nil
}
