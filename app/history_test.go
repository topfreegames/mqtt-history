// mqtt-history
// https://github.com/topfreegames/mqtt-history
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app_test

import (
	"bytes"
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
	. "github.com/topfreegames/mqtt-history/app"
	"github.com/topfreegames/mqtt-history/es"
//	"github.com/topfreegames/mqtt-history/redisclient"
  "github.com/topfreegames/mqtt-history/mongoclient"
	. "github.com/topfreegames/mqtt-history/testing"
  "gopkg.in/mgo.v2"
  "github.com/spf13/viper"
)

func refreshIndex() {
	_, err := http.Post("http://localhost:9123/_refresh", "application/json", bytes.NewBufferString("{}"))
	Expect(err).To(BeNil())
}

func msToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}

func TestHistoryHandler(t *testing.T) {
	g := Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("History", func() {
		esclient := es.GetESClient()

		g.BeforeEach(func() {
			refreshIndex()
		})

		g.Describe("History Handler", func() {
			g.It("It should return 401 if the user is not authorized into the topic", func() {
				a := GetDefaultTestApp()
				testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/history/chat/test_%s?userid=test:test", testId)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})
			
      g.It("It should return 200 if user is unauthorized into the topic but anonymous is enabled", func() {
        viper.Set("mongo.allow_anonymous", true)
        a := GetDefaultTestApp()
				testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/history/chat/test_%s?userid=test:test", testId)
				status, _ := Get(a, path, t)
        viper.Set("mongo.allow_anonymous", false)
        g.Assert(status).Equal(http.StatusOK)
      })

      g.It("It should return 200 if the user is authorized into the topic in mongo", func() {
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)

        Expect(err).To(BeNil())

        testMessage := Message{
					Timestamp: time.Now(),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}
				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())

				refreshIndex()
				path := fmt.Sprintf("/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

      })
     
			g.It("It should return 200 and [] if the user is authorized into the topic and there are no messages", func() {
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)

        Expect(err).To(BeNil())

				refreshIndex()
				path := fmt.Sprintf("/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
			})

      g.It("Should retrieve 1 message from history when topic matches wildcard", func() {
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)
        topics = append(topics, "chat/+")

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)

        Expect(err).To(BeNil())

				testMessage := Message{
					Timestamp: time.Now(),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}
				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())

				refreshIndex()
				path := fmt.Sprintf("/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
			})
		})

		g.Describe("History Since Handler", func() {
			g.It("It should return 401 if the user is not authorized into the topic", func() {
				a := GetDefaultTestApp()
				testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/historysince/chat/test_%s?userid=test:test", testId)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 if the user is authorized into the topic", func() {
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)

        Expect(err).To(BeNil())

				testMessage := Message{
					Timestamp: time.Now(),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())

				refreshIndex()

				path := fmt.Sprintf("/historysince/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
			})

			g.It("It should return 200 and [] if the user is authorized into the topic and there are no messages", func() {
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
        Expect(err).To(BeNil())

				refreshIndex()
				path := fmt.Sprintf("/historysince/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
			})

			g.It("It should return 200 if the user is authorized into the topic", func() {
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
        Expect(err).To(BeNil())

				testMessage := Message{
					Timestamp: time.Now(),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}

				path := fmt.Sprintf(
					"/historysince/%s?userid=test:test&since=%d",
					topic, (time.Now().UnixNano() / 1000000000), // now
				)
				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())

				// Update indexes
				refreshIndex()

				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				Expect(len(messages)).To(Equal(1))
				var message Message
				for i := 0; i < len(messages); i++ {
					message = messages[i]
					Expect(message.Topic).To(Equal(topic))
				}
			})

			g.It("It should return 200 if the user is authorized into the topic and since is in the future", func() {
        g.Timeout(10 * time.Second)
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
        Expect(err).To(BeNil())

				now := time.Now().UnixNano() / 1000000
				testMessage := Message{}
				second := int64(1000)
				baseTime := now - (second * 70)
				for i := 0; i < 150; i++ {
					messageTime := baseTime + 1*second
					testMessage = Message{
						Timestamp: msToTime(messageTime),
						Payload:   "{\"test1\":\"test2\"}",
						Topic:     topic,
					}
					_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
					Expect(err).To(BeNil())
				}

				// Update indexes
				refreshIndex()

				path := fmt.Sprintf(
					"/historysince/%s?userid=test:test&since=%d",
					topic, ((time.Now().UnixNano() / 100000000) * 200), // now
				)

				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				Expect(len(messages)).To(Equal(100))
				var message Message
				for i := 0; i < len(messages); i++ {
					message = messages[i]
					Expect(message.Topic).To(Equal(topic))
				}
			})

			g.It("It should return 200 if the user is authorized into the topic and since is negative", func() {
        g.Timeout(10 * time.Second)
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
        Expect(err).To(BeNil())

				now := time.Now().UnixNano() / 1000000
				testMessage := Message{}
				second := int64(1000)
				baseTime := now - (second * 70)
				for i := 0; i < 150; i++ {
					messageTime := baseTime + 1*second
					testMessage = Message{
						Timestamp: msToTime(messageTime),
						Payload:   "{\"test1\":\"test2\"}",
						Topic:     topic,
					}
					_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
					Expect(err).To(BeNil())
				}

				// Update indexes
				refreshIndex()

				path := fmt.Sprintf("/historysince/%s?userid=test:test&since=-1&limit=100", topic)

				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				Expect(len(messages)).To(Equal(100))
				var message Message
				for i := 0; i < len(messages); i++ {
					message = messages[i]
					Expect(message.Topic).To(Equal(topic))
				}
			})

			g.It("Should retrieve 10 messages when limit is 10 and the history size is greater than this", func() {
        a := GetDefaultTestApp()
        testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
        topic := fmt.Sprintf("chat/test_%s", testId)

        var topics []string
        topics = append(topics, topic)

        query := func(c *mgo.Collection) error {
          fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
          return fn
        }

        err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
        Expect(err).To(BeNil())

				now := time.Now().UnixNano() / 1000000
				testMessage := Message{}
				second := int64(1000)
				baseTime := now - (second * 70)
				for i := 0; i <= 30; i++ {
					messageTime := baseTime + 1*second
					testMessage = Message{
						Timestamp: msToTime(messageTime),
						Payload:   "{\"test1\":\"test2\"}",
						Topic:     topic,
					}
					_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
					Expect(err).To(BeNil())
				}

				// Update indexes
				refreshIndex()

				path := fmt.Sprintf(
					"/historysince/%s?userid=test:test&since=%d&limit=%d&from=%d",
					topic, baseTime/1000, 10, 0,
				)

				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []Message
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				Expect(len(messages)).To(Equal(10))
				var message Message
				for i := 0; i < len(messages); i++ {
					message = messages[i]
					Expect(message.Topic).To(Equal(topic))
				}
			})
		})

		g.It("Should retrieve only messages from the exact topic", func() {
      a := GetDefaultTestApp()
      testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
      topic := fmt.Sprintf("chat/test_%s", testId)

      var topics []string
      topics = append(topics, topic)

      query := func(c *mgo.Collection) error {
        fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
        return fn
      }

      err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
      Expect(err).To(BeNil())

			now := time.Now().UnixNano() / 1000000
			testMessage := Message{}
			second := int64(1000)
			baseTime := now - (second * 70)

			messageTime := baseTime + 1*second
			testMessage = Message{
				Timestamp: msToTime(messageTime),
				Payload:   "{\"test1\":\"test2\"}",
				Topic:     topic,
			}
			_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
			Expect(err).To(BeNil())

			messageTime = baseTime + 1*second
			testMessage = Message{
				Timestamp: msToTime(messageTime),
				Payload:   "{\"test1\":\"test2\"}",
				Topic:     fmt.Sprintf("%s/moremore", topic),
			}
			_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
			Expect(err).To(BeNil())

			// Update indexes
			refreshIndex()

			path := fmt.Sprintf(
				"/historysince/%s?userid=test:test&since=%d&limit=%d&from=%d",
				topic, baseTime/1000, 10, 0,
			)

			status, body := Get(a, path, t)
			g.Assert(status).Equal(http.StatusOK)

			var messages []Message
			err = json.Unmarshal([]byte(body), &messages)
			Expect(err).To(BeNil())
			Expect(len(messages)).To(Equal(1))
			var message Message
			for i := 0; i < len(messages); i++ {
				message = messages[i]
				Expect(message.Topic).To(Equal(topic))
			}
		})

		g.It("Should retrieve all messages eve if limit is greater than the size of current history", func() {
      a := GetDefaultTestApp()
      testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
      topic := fmt.Sprintf("chat/test_%s", testId)

      var topics []string
      topics = append(topics, topic)

      query := func(c *mgo.Collection) error {
        fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
        return fn
      }

      err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
      Expect(err).To(BeNil())

			startTime := time.Now().UnixNano() / 1000000
			testMessage := Message{}
			for i := 0; i < 3; i++ {
				messageTime := time.Now().UnixNano() / 1000000
				testMessage = Message{
					Timestamp: msToTime(messageTime),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}
				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())
			}

			// Sorry bout this =/
			time.Sleep(200 * time.Millisecond)

			// Update indexes
			refreshIndex()

			path := fmt.Sprintf(
				"/historysince/%s?userid=test:test&since=%d&limit=%d&from=%d",
				topic, startTime/1000, 10, 0,
			)

			status, body := Get(a, path, t)
			g.Assert(status).Equal(http.StatusOK)

			var messages []Message
			err = json.Unmarshal([]byte(body), &messages)
			Expect(err).To(BeNil())
			Expect(len(messages)).To(Equal(3))
			var message Message
			for i := 0; i < len(messages); i++ {
				message = messages[i]
				Expect(message.Topic).To(Equal(topic))
			}
		})

		g.It("Should retrieve 1 message from history when limit is 1 and theres more than 1 message", func() {
      a := GetDefaultTestApp()
      testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
      topic := fmt.Sprintf("chat/test_%s", testId)

      var topics []string
      topics = append(topics, topic)

      query := func(c *mgo.Collection) error {
        fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
        return fn
      }

      err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
      Expect(err).To(BeNil())

			startTime := time.Now().UnixNano() / 1000000
			testMessage := Message{}
			for i := 0; i < 3; i++ {
				messageTime := time.Now().UnixNano() / 1000000
				testMessage = Message{
					Timestamp: msToTime(messageTime),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}
				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())
			}

			// Sorry bout this =/
			time.Sleep(200 * time.Millisecond)

			// Update indexes
			refreshIndex()

			path := fmt.Sprintf(
				"/historysince/%s?userid=test:test&since=%d&limit=%d&from=%d",
				topic, startTime/1000, 1, 0,
			)

			status, body := Get(a, path, t)
			g.Assert(status).Equal(http.StatusOK)

			var messages []Message
			err = json.Unmarshal([]byte(body), &messages)
			Expect(err).To(BeNil())
			Expect(len(messages)).To(Equal(1))

			var message Message
			for i := 0; i < len(messages); i++ {
				message = messages[i]
				Expect(message.Topic).To(Equal(topic))
			}
		})

		g.It("Should retrieve 1 message from history when topic matches wildcard", func() {
      a := GetDefaultTestApp()
      testId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
      topic := fmt.Sprintf("chat/test_%s", testId)

      var topics []string
      topics = append(topics, topic)

      query := func(c *mgo.Collection) error {
        fn := c.Insert(&Acl{Username: "test:test", Pubsub: topics})
        return fn
      }

      err := mongoclient.GetCollection("mqtt", "mqtt_acl", query)
      Expect(err).To(BeNil())

			startTime := time.Now().UnixNano() / 1000000
			testMessage := Message{}
			for i := 0; i < 3; i++ {
				messageTime := time.Now().UnixNano() / 1000000
				testMessage = Message{
					Timestamp: msToTime(messageTime),
					Payload:   "{\"test1\":\"test2\"}",
					Topic:     topic,
				}
				_, err = esclient.Index().Index(GetChatIndex()).Type("message").BodyJson(testMessage).Do(context.TODO())
				Expect(err).To(BeNil())
			}

			// Sorry bout this =/
			time.Sleep(200 * time.Millisecond)

			// Update indexes
			refreshIndex()

			path := fmt.Sprintf(
				"/historysince/%s?userid=test:test&since=%d&limit=%d&from=%d",
				topic, startTime/1000, 1, 0,
			)

			status, body := Get(a, path, t)
			g.Assert(status).Equal(http.StatusOK)

			var messages []Message
			err = json.Unmarshal([]byte(body), &messages)
			Expect(err).To(BeNil())
			Expect(len(messages)).To(Equal(1))

			var message Message
			for i := 0; i < len(messages); i++ {
				message = messages[i]
				Expect(message.Topic).To(Equal(topic))
			}
		})
	})
} 
