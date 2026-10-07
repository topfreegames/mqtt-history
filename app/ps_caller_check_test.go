// mqtt-history
// https://github.com/topfreegames/mqtt-history
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app_test

import (
	"encoding/base64"
	"net/http"
	"os"
	"testing"

	goblin "github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/topfreegames/mqtt-history/app"
	. "github.com/topfreegames/mqtt-history/testing"
)

const psPath = "/ps/v2/history?topic=chat/ps_caller_check&initialDate=2022-01-01&finalDate=2022-01-02"

func basicAuth(user, password string) map[string]string {
	token := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
	return map[string]string{"Authorization": "Basic " + token}
}

func psCallerCount(result, caller string) float64 {
	return testutil.ToFloat64(app.PSCallerCheckTotal.WithLabelValues(result, caller))
}

func TestPSCallerCheck(t *testing.T) {
	g := goblin.Goblin(t)

	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("PS caller check", func() {
		g.Describe("mode off", func() {
			g.It("should pass a request with no credential and count nothing", func() {
				a := GetDefaultTestApp()
				before := psCallerCount("refused", "unknown")

				status, _ := Get(a, psPath, t)

				g.Assert(status).Equal(http.StatusOK)
				g.Assert(psCallerCount("refused", "unknown")).Equal(before)
			})
		})

		g.Describe("mode log", func() {
			g.It("should pass a refused request and count it", func() {
				a := GetTestAppWithConfig(map[string]interface{}{"ps.auth.mode": "log"})
				before := psCallerCount("refused", "unknown")

				status, _ := Get(a, psPath, t)

				g.Assert(status).Equal(http.StatusOK)
				g.Assert(psCallerCount("refused", "unknown")).Equal(before + 1)
			})

			g.It("should count an accepted credential with its caller", func() {
				a := GetTestAppWithConfig(map[string]interface{}{"ps.auth.mode": "log"})
				before := psCallerCount("credential", "test")

				status, _ := GetWithHeaders(a, psPath, basicAuth("test-user", "test-password"), t)

				g.Assert(status).Equal(http.StatusOK)
				g.Assert(psCallerCount("credential", "test")).Equal(before + 1)
			})
		})

		g.Describe("mode enforce", func() {
			var a *app.App
			g.Before(func() {
				a = GetTestAppWithConfig(map[string]interface{}{"ps.auth.mode": "enforce"})
			})

			g.It("should return 401 with no credential", func() {
				status, _ := Get(a, psPath, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("should return 401 with a wrong password", func() {
				status, _ := GetWithHeaders(a, psPath, basicAuth("test-user", "wrong"), t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("should return 401 with an unknown user", func() {
				status, _ := GetWithHeaders(a, psPath, basicAuth("other-user", "test-password"), t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("should return 200 with the right credential", func() {
				before := psCallerCount("credential", "test")

				status, _ := GetWithHeaders(a, psPath, basicAuth("test-user", "test-password"), t)

				g.Assert(status).Equal(http.StatusOK)
				g.Assert(psCallerCount("credential", "test")).Equal(before + 1)
			})

			g.It("should return 200 from a listed X-Real-IP", func() {
				before := psCallerCount("address", "testaddress")

				status, _ := GetWithHeaders(a, psPath, map[string]string{"X-Real-IP": "192.0.2.10"}, t)

				g.Assert(status).Equal(http.StatusOK)
				g.Assert(psCallerCount("address", "testaddress")).Equal(before + 1)
			})

			g.It("should return 401 from another X-Real-IP", func() {
				status, _ := GetWithHeaders(a, psPath, map[string]string{"X-Real-IP": "192.0.2.11"}, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("should return 401 with a listed X-Forwarded-For alone", func() {
				status, _ := GetWithHeaders(a, psPath, map[string]string{"X-Forwarded-For": "192.0.2.10"}, t)
				g.Assert(status).Equal(http.StatusUnauthorized)
			})

			g.It("should read a caller credential from the environment", func() {
				os.Setenv("MQTTHISTORY_PS_AUTH_CREDENTIALS_ENVCALLER", "env-user:env-password")
				defer os.Unsetenv("MQTTHISTORY_PS_AUTH_CREDENTIALS_ENVCALLER")
				envApp := GetTestAppWithConfig(map[string]interface{}{
					"ps.auth.mode":    "enforce",
					"ps.auth.callers": "test,envcaller",
				})

				status, _ := GetWithHeaders(envApp, psPath, basicAuth("env-user", "env-password"), t)

				g.Assert(status).Equal(http.StatusOK)
			})
		})

		g.Describe("malformed config", func() {
			g.It("should fail the start on an unknown mode", func() {
				Expect(func() {
					GetTestAppWithConfig(map[string]interface{}{"ps.auth.mode": "on"})
				}).To(Panic())
			})

			g.It("should fail the start when a listed caller has no credential", func() {
				Expect(func() {
					GetTestAppWithConfig(map[string]interface{}{
						"ps.auth.mode":    "log",
						"ps.auth.callers": "test,missing",
					})
				}).To(Panic())
			})

			g.It("should fail the start when a credential has no colon", func() {
				Expect(func() {
					GetTestAppWithConfig(map[string]interface{}{
						"ps.auth.mode":             "enforce",
						"ps.auth.credentials.test": "test-user",
					})
				}).To(Panic())
			})

			g.It("should fail the start when an address is not name=ip", func() {
				Expect(func() {
					GetTestAppWithConfig(map[string]interface{}{
						"ps.auth.mode":      "log",
						"ps.auth.addresses": "192.0.2.10",
					})
				}).To(Panic())
			})

			g.It("should fail the start with no caller and no address", func() {
				Expect(func() {
					GetTestAppWithConfig(map[string]interface{}{
						"ps.auth.mode":      "enforce",
						"ps.auth.callers":   "",
						"ps.auth.addresses": "",
					})
				}).To(Panic())
			})

			g.It("should start with a malformed entry in mode off", func() {
				Expect(func() {
					GetTestAppWithConfig(map[string]interface{}{"ps.auth.addresses": "192.0.2.10"})
				}).NotTo(Panic())
			})
		})
	})
}
