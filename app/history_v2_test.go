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
	"github.com/topfreegames/mqtt-history/models"
	. "github.com/topfreegames/mqtt-history/testing"
)

func TestHistoryV2Handler(t *testing.T) {
	g := goblin.Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("History2", func() {
		ctx := context.Background()
		a := GetDefaultTestApp()

		g.Describe("HistoryV2 Handler", func() {
			g.It("It should return 401 if the user is not authorized into the topic", func() {
				userID := fmt.Sprintf("test:%s", uuid.NewV4().String())
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/v2/history/chat/test_%s?userid=%s", testID, userID)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 if the user is authorized into the topic in mongo", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)

				err := AuthorizeTestUserInTopics(ctx, []string{topic})
				Expect(err).To(BeNil())

				err = InsertMongoMessages(ctx, []string{topic})
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/v2/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Payload["test 0"]).Equal("test 1")
				g.Assert(messages[0].Message).Equal("message 0")
			})

			g.It("It should return 200 and [] if the user is authorized into the topic and there are no messages", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)

				err := AuthorizeTestUserInTopics(ctx, []string{topic})
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/v2/history/%s?userid=test:test", topic)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err = json.Unmarshal([]byte(body), &messages)
				g.Assert(len(messages)).Equal(0)
				Expect(err).To(BeNil())
			})

			g.It("It should return 200 and the unblocked messages if the user is authorized into the topic ", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				userID := "test:test"

				err := AuthorizeTestUserInTopics(ctx, []string{topic})
				Expect(err).To(BeNil())

				err = InsertMongoMessagesWithParameters(ctx, []string{topic}, false)
				Expect(err).To(BeNil())

				err = InsertMongoMessagesWithParameters(ctx, []string{topic}, true)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/v2/history/%s?userid=%s&limit=1000", topic, userID)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2

				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Blocked).Equal(false)

			})

			g.It("It should return 200 and only blocked messages if the user is authorized into the topic", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				userID := "test:test"

				err := AuthorizeTestUserInTopics(ctx, []string{topic})
				Expect(err).To(BeNil())

				err = InsertMongoMessagesWithParameters(ctx, []string{topic}, false)
				Expect(err).To(BeNil())

				err = InsertMongoMessagesWithParameters(ctx, []string{topic}, true)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/v2/history/%s?userid=%s&limit=1000&isBlocked=true", topic, userID)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2

				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Blocked).Equal(true)

			})

			g.It("It should return 200 and only mensagens that are not blocked if the user is authorized into the topic but sent a wrong isBlocked flag", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				userID := "test:test"

				err := AuthorizeTestUserInTopics(ctx, []string{topic})
				Expect(err).To(BeNil())

				err = InsertMongoMessagesWithParameters(ctx, []string{topic}, false)
				Expect(err).To(BeNil())

				err = InsertMongoMessagesWithParameters(ctx, []string{topic}, true)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/v2/history/%s?userid=%s&limit=1000&isBlocked=wrongFlagHere", topic, userID)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2

				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Blocked).Equal(false)

			})
		})
	})
}
