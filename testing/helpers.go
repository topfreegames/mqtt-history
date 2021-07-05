// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/engine/standard"
	"github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/topfreegames/mqtt-history/app"
	"github.com/topfreegames/mqtt-history/models"
	"github.com/topfreegames/mqtt-history/mongoclient"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const cfgFile = "../config/test.yaml"

// GetDefaultTestApp retrieve a default app for testing purposes
func GetDefaultTestApp() *app.App {
	viper.SetConfigFile(cfgFile)
	app := app.GetApp("0.0.0.0", 8888, true, cfgFile)

	return app
}

// Get implements the GET http verb for testing purposes
func Get(app *app.App, url string, t *testing.T) (int, string) {
	return doRequest(app, "GET", url, "")
}

func doRequest(app *app.App, method, url, body string) (int, string) {
	app.Engine.SetHandler(app.API)
	ts := httptest.NewServer(app.Engine.(*standard.Server))
	defer ts.Close()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", ts.URL, url), reader)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	client := &http.Client{}
	res, err := client.Do(req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	if res != nil {
		defer res.Body.Close()
	}

	b, err := ioutil.ReadAll(res.Body)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return res.StatusCode, string(b)
}

func AuthorizeTestUserInTopics(ctx context.Context, topics []string) error {
	var acl []interface{}
	for _, topic := range topics {
		acl = append(acl, app.ACL{Username: "test:test", Pubsub: []string{topic}})
	}

	insertAuthCallback := func(c *mongo.Collection) error {
		var _, err = c.InsertMany(ctx, acl)
		return err
	}

	err := mongoclient.GetCollection("mqtt_acl", insertAuthCallback)
	return err
}

func InsertMongoMessages(ctx context.Context, topics []string) error {
	var messages []interface{}
	for i, topic := range topics {
		message := models.MessageV2{
			Id:             strconv.FormatInt(int64(i), 10),
			GameId:         "game test",
			PlayerId:       "test",
			Blocked:        false,
			ShouldModerate: true,
			Timestamp:      time.Now().AddDate(0, 0, -i).Unix(),
			Payload: bson.M{
				fmt.Sprintf("test %d", i): fmt.Sprintf("test %d", i+1),
			},
			Topic:    topic,
			Message:  fmt.Sprintf("message %d", i),
			Metadata: nil,
		}

		messages = append(messages, message)
	}

	// and given that the user has 2 messages stored in mongo
	insertMessagesCallback := func(c *mongo.Collection) error {
		_, err := c.InsertMany(ctx, messages)
		return err
	}

	messagesCollection := "messages"
	err := mongoclient.GetCollection(messagesCollection, insertMessagesCallback)
	return err
}
