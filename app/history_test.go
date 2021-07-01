// mqtt-history
// https://github.com/topfreegames/mqtt-history
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	goblin "github.com/franela/goblin"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/viper"
	. "github.com/topfreegames/mqtt-history/app"
	"github.com/topfreegames/mqtt-history/models"
	"github.com/topfreegames/mqtt-history/mongoclient"
	. "github.com/topfreegames/mqtt-history/testing"
	"go.mongodb.org/mongo-driver/mongo"
)

func msToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}

func TestHistoryHandler(t *testing.T) {
	g := goblin.Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("History", func() {
		ctx := context.Background()
		a := GetDefaultTestApp()

		g.AfterEach(func() {
			a.Defaults.MongoEnabled = false
		})

		g.Describe("History Handler", func() {
			g.It("It should return 401 if the user is not authorized into the topic", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/history/chat/test_%s?userid=test:test", testID)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 if user is unauthorized into the topic but anonymous is enabled", func() {
				viper.Set("mongo.allow_anonymous", true)
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/history/chat/test_%s?userid=test:test", testID)
				status, _ := Get(a, path, t)
				viper.Set("mongo.allow_anonymous", false)
				g.Assert(status).Equal(http.StatusOK)
			})

			g.It("It should return 200 if the user is authorized into the topic in mongo", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)

				var topics []string
				topics = append(topics, topic)

				query := func(c *mongo.Collection) error {
					_, err := c.InsertOne(ctx, ACL{Username: "test:test", Pubsub: topics})
					return err
				}

				err := mongoclient.GetCollection("mqtt_acl", query)

				Expect(err).To(BeNil())

				testMessage := models.Message{
					Timestamp: time.Now(),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				bucket := a.Bucket.Get(testMessage.Timestamp.Unix())
				err = a.Cassandra.InsertWithTTL(context.TODO(), testMessage.Topic, testMessage.Payload, bucket)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

			})

			g.It("It should return 200 if the user is authorized and mongo is used as message store", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)

				var topics []string
				topics = append(topics, topic)

				insertAuthCallback := func(c *mongo.Collection) error {
					_, err := c.InsertOne(ctx, ACL{Username: "test:test", Pubsub: topics})
					return err
				}

				err := mongoclient.GetCollection("mqtt_acl", insertAuthCallback)

				Expect(err).To(BeNil())

				testMessage := models.Message{
					Timestamp: time.Now().Add(-1 * time.Second),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				insertMessageCallback := func(c *mongo.Collection) error {
					_, err := c.InsertOne(ctx, testMessage)
					return err
				}

				messagesCollection := a.Config.GetString("mongo.messages.collection")
				err = mongoclient.GetCollection(messagesCollection, insertMessageCallback)
				Expect(err).To(BeNil())

				// enable mongo as message store
				a.Defaults.MongoEnabled = true

				path := fmt.Sprintf("/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
			})

			g.It("It should return 200 and [] if the user is authorized into the topic and there are no messages", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)

				var topics []string
				topics = append(topics, topic)

				query := func(c *mongo.Collection) error {
					_, err := c.InsertOne(ctx, ACL{Username: "test:test", Pubsub: topics})
					return err
				}

				err := mongoclient.GetCollection("mqtt_acl", query)

				Expect(err).To(BeNil())

				path := fmt.Sprintf("/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
			})

			g.It("Should retrieve 1 message from history when topic matches wildcard", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)

				var topics []string
				topics = append(topics, topic)
				topics = append(topics, "chat/+")

				query := func(c *mongo.Collection) error {
					_, err := c.InsertOne(ctx, ACL{Username: "test:test", Pubsub: topics})
					return err
				}

				err := mongoclient.GetCollection("mqtt_acl", query)

				Expect(err).To(BeNil())

				testMessage := models.Message{
					Timestamp: time.Now(),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				bucket := a.Bucket.Get(testMessage.Timestamp.Unix())
				err = a.Cassandra.InsertWithTTL(context.TODO(), testMessage.Topic, testMessage.Payload, bucket)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
			})
		})
	})
}
