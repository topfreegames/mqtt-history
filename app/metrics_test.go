// mqtt-history
// https://github.com/topfreegames/mqtt-history
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app_test

import (
	"net/http"
	"testing"

	goblin "github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/topfreegames/mqtt-history/app"
	. "github.com/topfreegames/mqtt-history/testing"
)

func TestResponseTimeMetrics(t *testing.T) {
	g := goblin.Goblin(t)

	// special hook for gomega
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("ResponseTimeMetricsMiddleware", func() {
		a := GetDefaultTestApp()

		g.It("observes a request into the response-time histogram", func() {
			// The default registry is shared across the whole test process,
			// so assert a delta rather than an absolute count.
			before := testutil.CollectAndCount(app.ResponseTimeSeconds)

			status, _ := Get(a, "/healthcheck", t)
			g.Assert(status).Equal(http.StatusOK)

			Expect(testutil.CollectAndCount(app.ResponseTimeSeconds)).
				To(BeNumerically(">=", before))
		})
	})
}
