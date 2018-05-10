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

	. "github.com/franela/goblin"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
	"github.com/topfreegames/extensions/mongo/interfaces"
	. "github.com/topfreegames/mqtt-history/app"
	"github.com/topfreegames/mqtt-history/models"
	"github.com/topfreegames/mqtt-history/mongoclient"
	. "github.com/topfreegames/mqtt-history/testing"
)

func TestHistoriesHandler(t *testing.T) {
	g := Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("Histories", func() {
		ctx := context.Background()
		a := GetDefaultTestApp()

		g.Describe("Histories Handler", func() {
			g.It("It should return 401 if the user is not authorized into the topics", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/history/chat/test_?userid=test:test&topics=%s", testID)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 if the user is authorized into the topics", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testID)
				topic2 := fmt.Sprintf("chat/test/%s", testID2)

				var topics, topics2 []string
				topics = append(topics, topic)
				topics2 = append(topics2, topic2)

				query := func(c interfaces.Collection) error {
					fn := c.Insert(&ACL{Username: "test:test", Pubsub: topics}, &ACL{Username: "test:test", Pubsub: topics2})
					return fn
				}

				err := mongoclient.GetCollection(ctx, "mqtt_acl", query)
				Expect(err).To(BeNil())

				testMessage := models.Message{
					Timestamp: time.Now().AddDate(0, 0, -1),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				testMessage2 := models.Message{
					Timestamp: time.Now(),
					Payload:   "{\"test3\":\"test4\"}",
					Topic:     topic2,
				}

				err = a.Cassandra.InsertWithTTL(ctx, testMessage.Topic, testMessage.Payload,
					testMessage.Timestamp)
				Expect(err).To(BeNil())
				err = a.Cassandra.InsertWithTTL(ctx, testMessage2.Topic, testMessage2.Payload,
					testMessage2.Timestamp)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(2)
				g.Assert(messages[0].Payload).Equal("{\"test1\":\"test2\"}")
				g.Assert(messages[1].Payload).Equal("{\"test3\":\"test4\"}")
			})

			g.It("It should return 200 if the user is authorized into at least one topic", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testID)
				topic2 := fmt.Sprintf("chat/test/%s", testID2)

				var topics, topics2 []string
				topics = append(topics, topic)
				topics2 = append(topics2, topic2)

				query := func(c interfaces.Collection) error {
					fn := c.Insert(&ACL{Username: "test:test", Pubsub: topics})
					return fn
				}

				err := mongoclient.GetCollection(ctx, "mqtt_acl", query)
				Expect(err).To(BeNil())

				testMessage := models.Message{
					Timestamp: time.Now().AddDate(0, 0, -1),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				testMessage2 := models.Message{
					Timestamp: time.Now(),
					Payload:   "{\"test3\":\"test4\"}",
					Topic:     topic2,
				}

				err = a.Cassandra.InsertWithTTL(ctx, testMessage.Topic, testMessage.Payload,
					testMessage.Timestamp)
				Expect(err).To(BeNil())

				err = a.Cassandra.InsertWithTTL(ctx, testMessage2.Topic, testMessage2.Payload,
					testMessage2.Timestamp)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Payload).Equal("{\"test1\":\"test2\"}")
			})

			g.It("It should return 401 if the user is not authorized in any topic", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testID)
				topic2 := fmt.Sprintf("chat/test/%s", testID2)

				var topics []string
				//topics = append(topics, topic)
				//topics = append(topics, topic2)

				query := func(c interfaces.Collection) error {
					fn := c.Insert(&ACL{Username: "test:test", Pubsub: topics})
					return fn
				}

				err := mongoclient.GetCollection(ctx, "mqtt_acl", query)
				Expect(err).To(BeNil())

				testMessage := models.Message{
					Timestamp: time.Now().AddDate(0, 0, -1),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				testMessage2 := models.Message{
					Timestamp: time.Now(),
					Payload:   "{\"test3\":\"test4\"}",
					Topic:     topic2,
				}

				err = a.Cassandra.InsertWithTTL(ctx, testMessage.Topic, testMessage.Payload,
					testMessage.Timestamp)
				Expect(err).To(BeNil())

				err = a.Cassandra.InsertWithTTL(ctx, testMessage2.Topic, testMessage2.Payload,
					testMessage2.Timestamp)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 if the user is authorized into the topics via wildcard", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testID)
				topic2 := fmt.Sprintf("chat/test/%s", testID2)

				var topics, topics2 []string
				topics = append(topics, topic)
				topics2 = append(topics2, topic2)

				query := func(c interfaces.Collection) error {
					fn := c.Insert(&ACL{Username: "test:test", Pubsub: topics}, &ACL{Username: "test:test", Pubsub: topics2})
					return fn
				}

				err := mongoclient.GetCollection(ctx, "mqtt_acl", query)
				Expect(err).To(BeNil())

				testMessage := models.Message{
					Timestamp: time.Now().AddDate(0, 0, -1),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				testMessage2 := models.Message{
					Timestamp: time.Now(),
					Payload:   "{\"test3\":\"test4\"}",
					Topic:     topic2,
				}

				err = a.Cassandra.InsertWithTTL(ctx, testMessage.Topic, testMessage.Payload,
					testMessage.Timestamp)
				Expect(err).To(BeNil())

				err = a.Cassandra.InsertWithTTL(ctx, testMessage2.Topic, testMessage2.Payload,
					testMessage2.Timestamp)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(2)
				g.Assert(messages[0].Payload).Equal("{\"test1\":\"test2\"}")
				g.Assert(messages[1].Payload).Equal("{\"test3\":\"test4\"}")
			})
		})
	})
}
