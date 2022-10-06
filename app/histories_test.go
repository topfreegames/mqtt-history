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

	goblin "github.com/franela/goblin"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
	. "github.com/topfreegames/mqtt-history/app"
	"github.com/topfreegames/mqtt-history/models"
	"github.com/topfreegames/mqtt-history/mongoclient"
	. "github.com/topfreegames/mqtt-history/testing"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestHistoriesHandler(t *testing.T) {
	g := goblin.Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("Histories", func() {
		ctx := context.Background()
		a := GetDefaultTestApp()

		g.Describe("Histories Handler", func() {

			g.It("It should return 401 if the user is not authorized into the topics", func() {
				userID := fmt.Sprintf("test:%s", uuid.NewV4().String())
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/v2/histories/chat/test?userid=%s&topics=%s,%s", userID, testID, testID2)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 if the user is authorized into the topics and mongo is used as message storage", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testID)
				topic2 := fmt.Sprintf("chat/test/%s", testID2)

				authorizedTopics := []string{topic, topic2}
				err := AuthorizeTestUserInTopics(ctx, authorizedTopics)
				Expect(err).To(BeNil())

				err = InsertMongoMessages(ctx, authorizedTopics)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, body := Get(a, path, t)

				// then the messages should be returned when requested via /histories
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(2)
				g.Assert(messages[0].Payload).Equal("{\"test 0\":\"test 1\"}")
				g.Assert(messages[1].Payload).Equal("{\"test 1\":\"test 2\"}")
			})

			g.It("It should return 200 if the user is authorized into at least one topic", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testID)
				topic2 := fmt.Sprintf("chat/test/%s", testID2)

				authorizedTopics := []string{topic}
				err := AuthorizeTestUserInTopics(ctx, authorizedTopics)
				Expect(err).To(BeNil())

				err = InsertMongoMessages(ctx, []string{topic, topic2})
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Payload).Equal("{\"test 0\":\"test 1\"}")
			})

			g.It("It should return 401 if the user is not authorized in any topic", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testID)
				topic2 := fmt.Sprintf("chat/test/%s", testID2)

				var topics []string

				query := func(c *mongo.Collection) error {
					_, err := c.InsertOne(ctx, ACL{Username: "test:test", Pubsub: topics})
					return err
				}

				mongoCollection, err := mongoclient.GetCollection(ctx, "mqtt_acl")
				Expect(err).To(BeNil())

				err = query(mongoCollection)
				Expect(err).To(BeNil())

				err = InsertMongoMessages(ctx, []string{topic, topic2})
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

				authorizedTopics := []string{topic, topic2}
				err := AuthorizeTestUserInTopics(ctx, authorizedTopics)
				Expect(err).To(BeNil())

				err = InsertMongoMessages(ctx, []string{topic, topic2})
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(2)
			})
		})
	})
}
