// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2017 Top Free Games <backend@tfgco.com>

package mongoclient_test

import (
	"encoding/json"
	"testing"
	"time"

	goblin "github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"github.com/topfreegames/mqtt-history/models"
	"github.com/topfreegames/mqtt-history/mongoclient"
)

func TestConvertMessageV2ToMessage(t *testing.T) {
	g := goblin.Goblin(t)

	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("ConvertMessageV2ToMessage", func() {
		g.It("It should read a timestamp in seconds as the same instant", func() {
			sent := time.Now().Truncate(time.Second)

			message := mongoclient.ConvertMessageV2ToMessage(&models.MessageV2{
				Timestamp: sent.Unix(),
				Topic:     "chat/test",
			})

			g.Assert(message.Timestamp.Unix()).Equal(sent.Unix())
		})

		g.It("It should read a timestamp in milliseconds as the same instant", func() {
			sent := time.Now().Truncate(time.Millisecond)

			message := mongoclient.ConvertMessageV2ToMessage(&models.MessageV2{
				Timestamp: sent.UnixMilli(),
				Topic:     "chat/test",
			})

			g.Assert(message.Timestamp.UnixMilli()).Equal(sent.UnixMilli())
		})

		g.It("It should marshal a message that carries a millisecond timestamp", func() {
			message := mongoclient.ConvertMessageV2ToMessage(&models.MessageV2{
				Timestamp: time.Now().UnixMilli(),
				Topic:     "chat/test",
			})

			_, err := json.Marshal(message)
			Expect(err).To(BeNil())
		})
	})
}
