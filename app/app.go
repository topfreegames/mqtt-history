// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package app

import (
	"fmt"
	"os"

	"github.com/getsentry/raven-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine"
	"github.com/labstack/echo/engine/standard"
	newrelic "github.com/newrelic/go-agent"
	"github.com/spf13/viper"
	"github.com/topfreegames/mqtt-history/logger"
	"github.com/topfreegames/mqtt-history/redisclient"
	"github.com/uber-go/zap"
)

// App is the struct that defines the application
type App struct {
	Debug       bool
	Port        int
	Host        string
	API         *echo.Echo
	Engine      engine.Server
	RedisClient *redisclient.RedisClient
	NewRelic    newrelic.Application
}

// GetApp creates an app given the parameters
func GetApp(host string, port int, debug bool) *App {
	logger.SetupLogger(viper.GetString("logger.level"))
	logger.Logger.Debug(
		fmt.Sprintf("Starting app with host: %s, port: %d", host, port))
	app := &App{
		Host:  host,
		Port:  port,
		Debug: debug,
	}
	app.Configure()
	return app
}

// Configure configures the application
func (app *App) Configure() {
	app.setConfigurationDefaults()
	app.configureSentry()

	app.configureNewRelic()

	app.loadConfiguration()
	app.configureApplication()
}

func (app *App) configureNewRelic() {
	newRelicKey := viper.GetString("newrelic.key")
	config := newrelic.NewConfig("mqtt-history", newRelicKey)
	if newRelicKey == "" {
		logger.Logger.Info("New Relic is not enabled..")
		config.Enabled = false
	}
	nr, err := newrelic.NewApplication(config)
	if err != nil {
		logger.Logger.Error("Failed to initialize New Relic.", zap.Error(err))
		panic(fmt.Sprintf("Could not initialize New Relic, err: %s", err))
	}

	app.NewRelic = nr
	logger.Logger.Info("Initialized New Relic successfully.")
}

func (app *App) setConfigurationDefaults() {
	viper.SetDefault("healthcheck.workingText", "WORKING")
}

func (app *App) loadConfiguration() {
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		logger.Logger.Debug("Config file read successfully")
	} else {
		panic(fmt.Sprintf("Could not load configuration file, err: %s", err))
	}
}

func (app *App) configureSentry() {
	sentryURL := viper.GetString("sentry.url")
	logger.Logger.Info(fmt.Sprintf("Configuring sentry with URL %s", sentryURL))
	raven.SetDSN(sentryURL)
}

func (app *App) configureApplication() {
	app.Engine = standard.New(fmt.Sprintf("%s:%d", app.Host, app.Port))
	app.API = echo.New()
	a := app.API
	_, w, _ := os.Pipe()
	a.SetLogOutput(w)
	a.Use(NewLoggerMiddleware(zap.New(
		zap.NewJSONEncoder(),
	)).Serve)
	a.Use(NewSentryMiddleware(app).Serve)
	a.Use(VersionMiddleware)
	a.Use(NewRecoveryMiddleware(app.OnErrorHandler).Serve)
	a.Use(NewNewRelicMiddleware(app, zap.New(
		zap.NewJSONEncoder(),
	)).Serve)

	// Routes
	a.Get("/healthcheck", HealthCheckHandler(app))
	a.Get("/historysince/*", HistorySinceHandler(app))
	a.Get("/history/*", HistoryHandler(app))
	a.Get("/histories/*", HistoriesHandler(app))
	a.Get("/:other", NotFoundHandler(app))

	app.RedisClient = redisclient.GetRedisClient(viper.GetString("redis.host"), viper.GetInt("redis.port"), viper.GetString("redis.password"))
}

//OnErrorHandler handles application panics
func (app *App) OnErrorHandler(err interface{}, stack []byte) {
	logger.Logger.Error(err)

	var e error
	switch err.(type) {
	case error:
		e = err.(error)
	default:
		e = fmt.Errorf("%v", err)
	}

	tags := map[string]string{
		"source": "app",
		"type":   "panic",
	}
	raven.CaptureError(e, tags)
}

// Start starts the application
func (app *App) Start() {
	app.API.Run(app.Engine)
}
