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
	"github.com/topfreegames/mqtt-history/models"
	. "github.com/topfreegames/mqtt-history/testing"
)

func TestHistoryV2PSHandler(t *testing.T) {
	g := goblin.Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("History2", func() {
		ctx := context.Background()
		a := GetDefaultTestApp()

		g.Describe("HistoryV2PS Handler", func() {

			g.It("It should return 200 and [] if there are no messages from the player on the informed topic", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				playerID := "test"

				path := fmt.Sprintf("/ps/v2/history?topic=%s&playerId=%s&initialDate=2022-01-01&finalDate=2022-12-01", topic, playerID)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.Message
				err := json.Unmarshal([]byte(body), &messages)
				g.Assert(len(messages)).Equal(0)
				Expect(err).To(BeNil())
			})

			g.It("It should return 200 and unblocked messages from the given player,topic and date.", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				playerID := "test"
				finalDate := strings.Split(time.Now().AddDate(0, 0, 1).UTC().String(), " ")[0]

				err := InsertMongoMessagesWithParameters(ctx, []string{topic}, false)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/ps/v2/history?topic=%s&playerId=%s&initialDate=2022-01-01&finalDate=%s", topic, playerID, finalDate)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2

				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Blocked).Equal(false)

			})

			g.It("It should return 200 and unblocked messages from the given topic and date.", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				finalDate := strings.Split(time.Now().AddDate(0, 0, 1).UTC().String(), " ")[0]

				err := InsertMongoMessagesWithParameters(ctx, []string{topic}, false)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/ps/v2/history?topic=%s&initialDate=2022-01-01&finalDate=%s", topic, finalDate)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2

				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Blocked).Equal(false)

			})

			g.It("It should return 422 and an error message if the date parameter is being missed", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				playerID := "test"

				err := InsertMongoMessagesWithParameters(ctx, []string{topic}, false)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/ps/v2/history?topic=%s&playerId=%s", topic, playerID)
				status, _ := Get(a, path, t)
				g.Assert(status).Equal(http.StatusUnprocessableEntity)

			})

			g.It("It should return 200 and the messages if the topic parameter is being missed", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				playerID := "test"
				finalDate := strings.Split(time.Now().AddDate(0, 0, 1).UTC().String(), " ")[0]

				err := InsertMongoMessagesWithParameters(ctx, []string{topic}, false)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/ps/v2/history?playerId=%s&initialDate=2022-01-01&finalDate=%s", playerID, finalDate)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2

				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

				g.Assert(len(messages)).Equal(10)
				g.Assert(messages[0].Blocked).Equal(false)

			})

			g.It("It should return 200 and only blocked messages from the given player,topic and date.", func() {
				testID := strings.Replace(uuid.NewV4().String(), "-", "", -1)
				topic := fmt.Sprintf("chat/test_%s", testID)
				playerID := "test"
				finalDate := strings.Split(time.Now().AddDate(0, 0, 1).UTC().String(), " ")[0]

				err := AuthorizeTestUserInTopics(ctx, []string{topic})
				Expect(err).To(BeNil())

				err = InsertMongoMessagesWithParameters(ctx, []string{topic}, false)
				Expect(err).To(BeNil())

				err = InsertMongoMessagesWithParameters(ctx, []string{topic}, true)
				Expect(err).To(BeNil())

				path := fmt.Sprintf("/ps/v2/history?topic=%s&playerId=%s&initialDate=2022-01-01&finalDate=%s&isBlocked=true", topic, playerID, finalDate)
				status, body := Get(a, path, t)
				g.Assert(status).Equal(http.StatusOK)

				var messages []models.MessageV2

				err = json.Unmarshal([]byte(body), &messages)
				Expect(err).To(BeNil())

				g.Assert(len(messages)).Equal(1)
				g.Assert(messages[0].Blocked).Equal(true)

			})
		})
	})
}
