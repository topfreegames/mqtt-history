// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2017 Top Free Games <backend@tfgco.com>

package mongoclient

import (
	"context"
	netUrl "net/url"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/spf13/viper"
	"github.com/topfreegames/mqtt-history/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client   *mongo.Client
	user     string
	database string
	once     sync.Once
)

func mongoSession(ctx context.Context) (*mongo.Client, error) {
	var err error

	once.Do(func() {
		config := viper.GetViper()
		url := config.GetString("mongo.host")
		database = config.GetString("mongo.database")
		config.SetDefault("mongo.connectionTimeout", 10)
		connectionTimeout := time.Duration(config.GetInt("mongo.connectionTimeout")) * time.Second

		var urlStruct *netUrl.URL
		urlStruct, err = netUrl.Parse(url)
		if err != nil {
			logger.Logger.Error("Invalid connection URL for MongoDB", err)
		}
		user = urlStruct.User.Username()

		// Avoid logging password
		urlToLog := url
		pass, isPasswordSet := urlStruct.User.Password()
		if isPasswordSet {
			urlToLog = strings.Replace(urlToLog, pass, "<hidden>", -1)
		}
		logger.Logger.Infof("Connecting to MongoDB at '%s'", urlToLog)

		client, err = mongo.Connect(
			ctx,
			options.
				Client().
				ApplyURI(url).
				SetConnectTimeout(connectionTimeout))
	})

	if err != nil {
		return nil, err
	}
	return client, nil
}

// GetCollection returns a collection from the database
func GetCollection(ctx context.Context, collection string) (*mongo.Collection, error) {
	mongoDB, err := mongoSession(ctx)
	if err != nil {
		return nil, err
	}
	// staleness: check how old the data is before reading from Secondary replicas
	secondaryPreferredOpts := readpref.WithMaxStaleness(90 * time.Second)
	// secondaryPreferred: prefer reading from Secondary replicas, falling back to the primary if needed
	secondaryPreferred := readpref.SecondaryPreferred(secondaryPreferredOpts)
	dbOpts := options.Database().
		SetReadPreference(secondaryPreferred)

	return mongoDB.Database(database, dbOpts).Collection(collection), nil
}
