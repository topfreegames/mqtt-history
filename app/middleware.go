package app

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/topfreegames/mqtt-history/logger"
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

		//all except latency to string
		var ip, method, path string
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
			gameID = metricTagsMap["gameID"].(string)
		}

		// gameID := c.Get("metricTagsMap")
		// gameID = gameID.(map[string]interface{})["gameID"]

		//no time.Since in order to format it well after
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
		reqLog.Info("Request successful.")

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

// NewJaegerMiddleware create a new middleware to instrument traces.
func NewJaegerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tracer := opentracing.GlobalTracer()

			request := c.Request()
			method := request.Method()
			url := request.URL()
			header := getCarrier(request)
			parent, err := tracer.Extract(opentracing.HTTPHeaders, header)
			if err != nil && err != opentracing.ErrSpanContextNotFound {
				logger.Logger.Errorf(
					"Could no extract parent trace from incoming request. Method: %s. Path: %s.",
					method,
					url.Path(),
					err,
				)
			}

			operationName := fmt.Sprintf("HTTP %s %s", method, c.Path())
			reference := opentracing.ChildOf(parent)
			tags := opentracing.Tags{
				"http.method":   method,
				"http.host":     request.Host(),
				"http.pathname": url.Path(),
				"http.query":    url.QueryString(),
				"span.kind":     "server",
			}
			span := opentracing.StartSpan(operationName, reference, tags)
			defer span.Finish()
			defer func(span opentracing.Span) {
				if err, ok := recover().(error); ok {
					span.SetTag("error", true)
					span.LogFields(
						log.String("event", "error"),
						log.String("message", "Panic serving request."),
						log.Error(err),
					)
					panic(err)
				}
			}(span)

			ctx := c.StdContext()
			ctx = opentracing.ContextWithSpan(ctx, span)
			c.SetStdContext(ctx)

			err = next(c)
			if err != nil {
				span.SetTag("error", true)
				span.LogFields(
					log.String("event", "error"),
					log.String("message", "Error serving request."),
					log.Error(err),
				)
			}

			response := c.Response()
			statusCode := response.Status()
			span.SetTag("http.status_code", statusCode)

			return err
		}
	}
}

func getCarrier(request engine.Request) opentracing.HTTPHeadersCarrier {
	original := request.Header()
	copy := make(http.Header)
	for _, key := range original.Keys() {
		value := original.Get(key)
		copy.Set(key, value)
	}
	return opentracing.HTTPHeadersCarrier(copy)
}
