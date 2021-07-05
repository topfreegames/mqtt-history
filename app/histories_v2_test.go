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

func TestHistoriesV2Handler(t *testing.T) {
	g := goblin.Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("HistoriesV2", func() {
		ctx := context.Background()
		a := GetDefaultTestApp()

		g.Describe("HistoriesV2 Handler", func() {

			g.It("It should return 401 if the user is not authorized into the topics", func() {
				userID := fmt.Sprintf("test:%s", uuid.NewV4().String())
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				path := fmt.Sprintf("/v2/histories/chat/test?userid=%s&topics=%s,%s", userID, testID, testID2)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("It should return 200 and messages if the user is authorized into the topics", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				testID2 := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test/%s", testID)
				topic2 := fmt.Sprintf("chat/test/%s", testID2)
				topics := []string{topic, topic2}

				err := AuthorizeTestUserInTopics(ctx, topics)
				Expect(err).To(BeNil())

				err = InsertMongoMessages(ctx, topics)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/v2/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, body := Get(a, path, t)

				// then the messages should be returned when requested via /histories
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(2)
				g.Assert(messages[0].Payload["test 0"]).Equal("test 1")
				g.Assert(messages[0].Message).Equal("message 0")
				g.Assert(messages[1].Payload["test 1"]).Equal("test 2")
				g.Assert(messages[1].Message).Equal("message 1")
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

				path := fmt.Sprintf("/v2/histories/chat/test?userid=test:test&topics=%s,%s", testID, testID2)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2
				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())
				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Payload["test 0"]).Equal("test 1")
			})
		})
	})
}
