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
	"github.com/satori/go.uuid"
	. "github.com/topfreegames/mqtt-history/app"
	"github.com/topfreegames/mqtt-history/es"
	"github.com/topfreegames/mqtt-history/mongoclient"
	. "github.com/topfreegames/mqtt-history/testing"
  "gopkg.in/mgo.v2"
)

func TestHistoriesHandler(t *testing.T) {
	g := Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("Histories", func() {
		esclient := es.GetESClient()

		g.BeforeEach(func() {
			refreshIndex()
		})

		g.Describe("Histories Handler", func() {
			g.It("It should return 401 if the user is not authorized into the topics", func() {
				a := GetDefaultTestApp()
				testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/history/chat/test_?userid=test:test&topics=%s", testId)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 if the user is authorized into the topics", func() {
				a := GetDefaultTestApp()
				testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testId2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testId)
				topic2 := fmt.Sprintf("chat/test/%s", testId2)
        
        var topics, topics2 []string
        topics = append(topics, topic)
        topics2 = append(topics2, topic2)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics}, &Acl{Username: "test:test", Pubsub: topics2})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
				Expect(err).To(BeNil())

				testMessage := Message{
					Timestamp: time.Now().AddDate(0, 0, -1),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				testMessage2 := Message{
					Timestamp: time.Now(),
					Payload:   "{\"test3\":\"test4\"}",
					Topic:     topic2,
				}

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage2).Do(context.TODO())
				Expect(err).To(BeNil())

				refreshIndex()
				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testId, testId2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(messages[0].Payload).Equal("{\"test3\":\"test4\"}")
				g.Assert(messages[1].Payload).Equal("{\"test1\":\"test2\"}")
			})

			g.It("It should return 200 if the user is authorized into at least one topic", func() {
				a := GetDefaultTestApp()
				testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testId2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testId)
				topic2 := fmt.Sprintf("chat/test/%s", testId2)
				
        var topics, topics2 []string
        topics = append(topics, topic)
        topics2 = append(topics2, topic2)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
				Expect(err).To(BeNil())

				testMessage := Message{
					Timestamp: time.Now().AddDate(0, 0, -1),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				testMessage2 := Message{
					Timestamp: time.Now(),
					Payload:   "{\"test3\":\"test4\"}",
					Topic:     topic2,
				}

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage2).Do(context.TODO())
				Expect(err).To(BeNil())

				refreshIndex()
				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testId, testId2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(messages[0].Payload).Equal("{\"test1\":\"test2\"}")
				g.Assert(len(messages)).Equal(1)
			})

			g.It("It should return 401 if the user is not authorized in any topic", func() {
				a := GetDefaultTestApp()
				testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testId2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testId)
				topic2 := fmt.Sprintf("chat/test/%s", testId2)
				
        var topics []string
        //topics = append(topics, topic)
        //topics = append(topics, topic2)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
				Expect(err).To(BeNil())

				testMessage := Message{
					Timestamp: time.Now().AddDate(0, 0, -1),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				testMessage2 := Message{
					Timestamp: time.Now(),
					Payload:   "{\"test3\":\"test4\"}",
					Topic:     topic2,
				}

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage2).Do(context.TODO())
				Expect(err).To(BeNil())

				refreshIndex()
				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testId, testId2)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 if the user is authorized into the topics via wildcard", func() {
				a := GetDefaultTestApp()
				testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testId2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testId)
				topic2 := fmt.Sprintf("chat/test/%s", testId2)

        var topics, topics2 []string
        topics = append(topics, topic)
        topics2 = append(topics2, topic2)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics}, &Acl{Username: "test:test", Pubsub: topics2})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
				Expect(err).To(BeNil())

				testMessage := Message{
					Timestamp: time.Now().AddDate(0, 0, -1),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				testMessage2 := Message{
					Timestamp: time.Now(),
					Payload:   "{\"test3\":\"test4\"}",
					Topic:     topic2,
				}

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage2).Do(context.TODO())
				Expect(err).To(BeNil())

				refreshIndex()
				path := fmt.Sprintf("/histories/chat/test?userid=test:test&topics=%s,%s", testId, testId2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(messages[0].Payload).Equal("{\"test3\":\"test4\"}")
				g.Assert(messages[1].Payload).Equal("{\"test1\":\"test2\"}")
			})
		})
	})
}
