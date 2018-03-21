package es

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/olivere/elastic.v5"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
	eelastic "github.com/topfreegames/extensions/elastic"
	"github.com/topfreegames/mqtt-history/logger"
)

var esclient *elastic.Client
var once sync.Once

type EsLogger struct {
	Logger *logging.Logger
}

func (e *EsLogger) Printf(format string, v ...interface{}) {
	e.Logger.Debugf(format, v)
}

// GetESClient returns the elasticsearch client with the given configs
func GetESClient() *elastic.Client {
	once.Do(func() {
		configure()
	})
	return esclient
}

func configure() {
	setConfigurationDefaults()
	configureESClient()
}

func setConfigurationDefaults() {
	viper.SetDefault("elasticsearch.host", "http://localhost:9200")
	viper.SetDefault("elasticsearch.sniff", false)
	viper.SetDefault("elasticsearch.indexMappings", map[string]string{})
}

func configureESClient() {
	logger.Logger.Debug(fmt.Sprintf("Connecting to elasticsearch @ %s",
		viper.GetString("elasticsearch.host")))

	client, err := eelastic.NewClient(
		elastic.SetURL(viper.GetString("elasticsearch.host")),
		elastic.SetSniff(viper.GetBool("elasticsearch.sniff")),
		elastic.SetTraceLog(&EsLogger{Logger: logger.Logger}),
	)
	if err != nil {
		logger.Logger.Fatal(fmt.Sprintf("Failed to connect to elasticsearch! err: %v", err))
	}

	logger.Logger.Info(fmt.Sprintf("Successfully connected to elasticsearch @ %s",
		viper.GetString("elasticsearch.host")))
	logger.Logger.Debug("Creating index chat into ES")

	indexes := viper.GetStringMapString("elasticsearch.indexMappings")
	for index, mappings := range indexes {
		_, err = client.CreateIndex(index).Body(mappings).Do(context.TODO())
		if err != nil {
			if strings.Contains(err.Error(), "index_already_exists_exception") || strings.Contains(err.Error(), "IndexAlreadyExistsException") ||
				strings.Contains(err.Error(), "already exists as alias") {
				logger.Logger.Warning(fmt.Sprintf("Index %s already exists into ES! Ignoring creation...", index))
			} else {
				logger.Logger.Error(fmt.Sprintf("Failed to create index %s into ES, err: %s", index, err))
				os.Exit(1)
			}
		} else {
			logger.Logger.Debug(fmt.Sprintf("Sucessfully created index %s into ES", index))
		}
	}
	esclient = client
}
