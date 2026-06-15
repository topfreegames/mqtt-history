package app

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/labstack/echo"
	"github.com/topfreegames/extensions/middleware"
	"github.com/uber-go/zap"
)

// VersionMiddleware automatically adds a version header to response
func VersionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderServer, fmt.Sprintf("mqtt-history/v%s", VERSION))
		return next(c)
	}
}

// NewRecoveryMiddleware returns a configured middleware
func NewRecoveryMiddleware(onError func(interface{}, []byte)) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		OnError: onError,
	}
}

// RecoveryMiddleware recovers from errors in Echo
type RecoveryMiddleware struct {
	OnError func(interface{}, []byte)
}

// Serve executes on error handler when errors happen
func (r RecoveryMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if err := recover(); err != nil {
				if r.OnError != nil {
					r.OnError(err, debug.Stack())
				}

				if eError, ok := err.(error); ok {
					c.Error(eError)
				} else {
					eError = fmt.Errorf(fmt.Sprintf("%v", err))
					c.Error(eError)
				}
			}
		}()
		return next(c)
	}
}

// LoggerMiddleware is responsible for logging to Zap all requests
type LoggerMiddleware struct {
	Logger zap.Logger
}

// Serve serves the middleware
func (l LoggerMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := l.Logger.With(
			zap.String("source", "request"),
		)

		// all except latency to string
		var ip, method, path, gameID string
		var status int
		var latency time.Duration
		var startTime, endTime time.Time

		path = c.Path()
		method = c.Request().Method()

		startTime = time.Now()

		metricTagsMap := make(map[string]interface{})
		c.Set("metricTagsMap", metricTagsMap)

		result := next(c)

		if metricTagsMap, ok := c.Get("metricTagsMap").(map[string]interface{}); ok {
			gameID, _ = metricTagsMap["gameID"].(string)
		}

		// no time.Since in order to format it well after
		endTime = time.Now()
		latency = endTime.Sub(startTime)

		status = c.Response().Status()
		ip = c.Request().RemoteAddress()

		route := c.Get("route")
		if route == nil {
			log.Debug("Route does not have route set in ctx")
			return result
		}

		reqLog := log.With(
			zap.String("route", route.(string)),
			zap.Time("endTime", endTime),
			zap.Int("statusCode", status),
			zap.Duration("latency", latency),
			zap.String("ip", ip),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("gameID", gameID),
		)

		//request failed
		if status > 399 && status < 500 {
			reqLog.Warn("Request failed.")
			return result
		}

		//request is ok, but server failed
		if status > 499 {
			reqLog.Error("Response failed.")
			return result
		}
		//Everything went ok
		reqLog.Debug("Request successful.")
		return result

	}
}

// NewLoggerMiddleware returns the logger middleware
func NewLoggerMiddleware(theLogger zap.Logger) *LoggerMiddleware {
	l := &LoggerMiddleware{Logger: theLogger}
	return l
}

// SentryMiddleware is responsible for sending all exceptions to sentry
type SentryMiddleware struct{}

// Serve serves the middleware
func (s SentryMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			tags := map[string]string{
				"source": "app",
				"type":   "Internal server error",
				"url":    c.Request().URI(),
				"status": fmt.Sprintf("%d", c.Response().Status()),
			}
			raven.CaptureError(err, tags)
		}
		return err
	}
}

// NewSentryMiddleware returns a new sentry middleware
func NewSentryMiddleware() *SentryMiddleware {
	return &SentryMiddleware{}
}

// NewNewRelicMiddleware returns the logger middleware
func NewNewRelicMiddleware(app *App, theLogger zap.Logger) *NewRelicMiddleware {
	l := &NewRelicMiddleware{App: app, Logger: theLogger}
	return l
}

// NewRelicMiddleware is responsible for logging to Zap all requests
type NewRelicMiddleware struct {
	App    *App
	Logger zap.Logger
}

// Serve serves the middleware
func (nr *NewRelicMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		route := c.Path()
		txn := nr.App.NewRelic.StartTransaction(route, nil, nil)
		c.Set("txn", txn)
		defer func() {
			c.Set("txn", nil)
			txn.End()
		}()

		err := next(c)
		if err != nil {
			txn.NoticeError(err)
			return err
		}
		return nil
	}
}

// statsdMetricName is the legacy statsd timer name, kept for the transitional
// period while both statsd and Prometheus are emitted in parallel.
const statsdMetricName = "response_time_milliseconds"

// ResponseTimeMetricsMiddleware measures the response time of a route and
// records it into the Prometheus ResponseTimeSeconds histogram and, during the
// migration, the legacy DogStatsD timer.
type ResponseTimeMetricsMiddleware struct {
	// DDStatsD is the optional statsd client. When nil, no statsd timer is
	// emitted (Prometheus-only).
	DDStatsD *middleware.DogStatsD
	// EmitPrometheus controls whether the Prometheus histogram is observed.
	EmitPrometheus bool
}

// Serve measures the response time of a route and reports it to the enabled
// backends: the Prometheus histogram (seconds) and/or the statsd timer.
func (responseTimeMiddleware ResponseTimeMetricsMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		startTime := time.Now()
		result := next(c)
		status := c.Response().Status()
		route := c.Path()
		method := c.Request().Method()

		var gameID string
		if metricTagsMap, ok := c.Get("metricTagsMap").(map[string]interface{}); ok {
			gameID, _ = metricTagsMap["gameID"].(string)
		}

		timeUsed := time.Since(startTime)

		if responseTimeMiddleware.EmitPrometheus {
			ResponseTimeSeconds.WithLabelValues(
				route,
				method,
				fmt.Sprintf("%d", status),
				gameID,
			).Observe(timeUsed.Seconds())
		}

		if responseTimeMiddleware.DDStatsD != nil {
			tags := []string{
				fmt.Sprintf("route:%s", route),
				fmt.Sprintf("method:%s", method),
				fmt.Sprintf("status:%d", status),
				fmt.Sprintf("gameID:%v", gameID),
			}
			responseTimeMiddleware.DDStatsD.Timing(statsdMetricName, timeUsed, tags...)
		}

		return result
	}
}

// NewResponseTimeMetricsMiddleware returns a new ResponseTimeMetricsMiddleware.
// ddStatsD may be nil to disable statsd; emitPrometheus toggles the histogram.
func NewResponseTimeMetricsMiddleware(ddStatsD *middleware.DogStatsD, emitPrometheus bool) *ResponseTimeMetricsMiddleware {
	return &ResponseTimeMetricsMiddleware{
		DDStatsD:       ddStatsD,
		EmitPrometheus: emitPrometheus,
	}
}
