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
	"strings"

	newrelic "github.com/newrelic/go-agent"
	extechomiddleware "github.com/topfreegames/extensions/echo/middleware"
	extnethttpmiddleware "github.com/topfreegames/extensions/middleware"

	"github.com/getsentry/raven-go"
	"github.com/labstack/echo/engine"
	"github.com/labstack/echo/engine/standard"
	"github.com/spf13/viper"
	"github.com/topfreegames/extensions/echo"
	"github.com/topfreegames/mqtt-history/cassandra"
	"github.com/topfreegames/mqtt-history/models"
	"github.com/topfreegames/mqtt-history/mongoclient"
	"github.com/uber-go/zap"

	"github.com/uber/jaeger-client-go/config"
)

// App is the struct that defines the application
type App struct {
	Debug                bool
	Port                 int
	Host                 string
	API                  *echo.Echo
	Engine               engine.Server
	ConfigPath           string
	Config               *viper.Viper
	NewRelic             newrelic.Application
	NumberOfDaysToSearch int
	DDStatsD             *extnethttpmiddleware.DogStatsD
	Cassandra            cassandra.DataStore
	Defaults             *models.Defaults
	Bucket               *models.Bucket
	Logger               zap.Logger
}

// GetApp creates an app given the parameters
func GetApp(host string, port int, debug bool, configPath string) *App {
	logLevel := zap.InfoLevel
	err := logLevel.UnmarshalText([]byte(viper.GetString("logger.level")))
	if err != nil {
		panic(err)
	}
	logger := zap.New(zap.NewJSONEncoder(), logLevel)
	logger.Debug(
		"Starting app",
		zap.String("host", host),
		zap.Int("port", port),
	)
	app := &App{
		Host:       host,
		Port:       port,
		Config:     viper.GetViper(),
		ConfigPath: configPath,
		Debug:      debug,
		Logger:     logger,
	}
	app.Configure()
	return app
}

// Configure configures the application
func (app *App) Configure() {
	app.setConfigurationDefaults()
	app.loadConfiguration()
	app.configureDefaults()

	app.configureSentry()
	app.configureNewRelic()
	app.configureStatsD()
	app.configureJaeger()

	app.configureStorage()
	app.configureApplication()
}

func (app *App) configureBucket() {
	app.Bucket = models.NewBucket(app.Config)
}

func (app *App) configureStorage() {
	if app.Defaults.MongoEnabled {
		app.Defaults.LimitOfMessages = app.Config.GetInt64("mongo.messages.limit")
		mongoclient.SetLogger(app.Logger)
		return
	}

	app.configureBucket()
	if app.Defaults.CassandraEnabled {
		app.configureCassandra()
	}
}

func (app *App) configureDefaults() {
	app.Defaults = &models.Defaults{
		BucketQuantityOnSelect:  app.Config.GetInt("cassandra.bucket.quantity"),
		LimitOfMessages:         app.Config.GetInt64("cassandra.messages.limit"),
		MongoEnabled:            app.Config.GetBool("mongo.messages.enabled"),
		MongoMessagesCollection: app.Config.GetString("mongo.messages.collection"),
		CassandraEnabled:        app.Config.GetBool("cassandra.enabled"),
	}
}

func (app *App) configureCassandra() {
	app.Logger.Info("Connecting to Cassandra")
	cassandra, err := cassandra.GetCassandra(
		app.Logger,
		app.Config,
		app.DDStatsD,
	)
	if err != nil {
		app.Logger.Error("Failed to initialize Cassandra.", zap.Error(err))
		panic(fmt.Sprintf("Could not initialize Cassandra, err: %s", err))
	}

	app.Logger.Info("Initialized Cassandra successfully.")
	app.Cassandra = cassandra
}

func (app *App) configureNewRelic() {
	newRelicKey := app.Config.GetString("newrelic.key")
	config := newrelic.NewConfig("mqtt-history", newRelicKey)
	if newRelicKey == "" {
		app.Logger.Info("New Relic is not enabled..")
		config.Enabled = false
	}
	nr, err := newrelic.NewApplication(config)
	if err != nil {
		app.Logger.Error("Failed to initialize New Relic.", zap.Error(err))
		panic(fmt.Sprintf("Could not initialize New Relic, err: %s", err))
	}

	app.NewRelic = nr
	app.Logger.Info("Initialized New Relic successfully.")
}

func (app *App) configureStatsD() {
	app.Logger.Info("Starting DogStatsD..")
	ddstatsd, err := extnethttpmiddleware.NewDogStatsD(app.Config)
	if err != nil {
		app.Logger.Error("Failed to initialize DogStatsD.", zap.Error(err))
		panic(fmt.Sprintf("Could not initialize DogStatsD, err: %s", err))
	}
	app.DDStatsD = ddstatsd
	app.Logger.Info("Initialized DogStatsD successfully.")
}

func (app *App) configureJaeger() {
	app.Logger.Info("Initializing Jaeger Global Tracer...")
	cfg, err := config.FromEnv()
	if err != nil {
		app.Logger.Error("Failed to load Jaeger config from env", zap.Error(err))
		return
	}
	if !cfg.Disabled {
		if cfg.ServiceName == "" {
			cfg.ServiceName = "mqtt-history"
		}
	}
	var configOptions []config.Option
	if cfg.Reporter.LogSpans {
		configOptions = append(configOptions, config.Logger(WrapZapLogger(app.Logger)))
	}
	if _, err := cfg.InitGlobalTracer("", configOptions...); err != nil {
		app.Logger.Error("Failed to initialize Jaeger.", zap.Error(err))
	} else {
		app.Logger.Info("Jaeger Global Tracer initialized successfully.")
	}
}

func (app *App) setConfigurationDefaults() {
	app.Config.SetDefault("healthcheck.workingText", "WORKING")
	app.Config.SetDefault("mongo.database", "mqtt")
}

func (app *App) loadConfiguration() {
	app.Logger.Info("ConfigPath: " + app.ConfigPath)

	app.Config.SetConfigType("yaml")
	app.Config.SetConfigFile(app.ConfigPath)
	app.Config.SetEnvPrefix("mqtthistory")
	app.Config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	app.Config.AutomaticEnv()

	if err := app.Config.ReadInConfig(); err == nil {
		app.Logger.Debug("Config file read successfully")
	} else {
		panic(fmt.Sprintf("Could not load configuration file, err: %s", err))
	}
}

func (app *App) configureSentry() {
	sentryURL := app.Config.GetString("sentry.url")
	app.Logger.Info(fmt.Sprintf("Configuring sentry with URL %s", sentryURL))
	raven.SetDSN(sentryURL)
}

func (app *App) configureApplication() {
	app.Engine = standard.New(fmt.Sprintf("%s:%d", app.Host, app.Port))
	app.NumberOfDaysToSearch = app.Config.GetInt("numberOfDaysToSearch")
	app.API = echo.New()
	a := app.API
	_, w, _ := os.Pipe()
	a.SetLogOutput(w)
	a.Use(NewLoggerMiddleware(app.Logger).Serve)
	a.Use(NewSentryMiddleware().Serve)
	a.Use(VersionMiddleware)
	a.Use(NewRecoveryMiddleware(app.OnErrorHandler).Serve)
	if app.Config.GetBool("extensions.dogstatsd.enabled") {
		a.Use(extechomiddleware.NewResponseTimeMetricsMiddleware(app.DDStatsD).Serve)
	}
	// Routes
	a.Get("/healthcheck", HealthCheckHandler(app))
	a.Get("/history/*", HistoryHandler(app))
	a.Get("/histories/*", HistoriesHandler(app))
	a.Get("/v2/history/*", HistoryV2Handler(app))
	a.Get("/v2/histories/*", HistoriesV2Handler(app))
	a.Get("/:other", NotFoundHandler(app))
	a.Get("/ps/v2/history*", HistoriesV2PSHandler(app))
}

// OnErrorHandler handles application panics
func (app *App) OnErrorHandler(err interface{}, stack []byte) {
	var e error
	switch err.(type) {
	case error:
		e = err.(error)
	default:
		e = fmt.Errorf("%v", err)
	}

	app.Logger.Error("Recovering from error", zap.Error(e))
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
