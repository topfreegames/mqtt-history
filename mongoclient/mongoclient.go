// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2017 Top Free Games <backend@tfgco.com>

package mongoclient

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/topfreegames/mqtt-history/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client   *mongo.Client
	database string
	once     sync.Once
)

func mongoSession() (*mongo.Client, error) {
	var err error

	once.Do(func() {
		config := viper.GetViper()
		url := config.GetString("mongo.host")
		database = config.GetString("mongo.database")

		const defaultTimeout = 10
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout*time.Second)
		defer cancel()

		logger.Logger.Infof("Connecting to MongoDB at '%s'", url)
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(url))
	})

	if err != nil {
		return nil, err
	}
	return client, nil
}

// GetCollection returns a collection from the database
func GetCollection(collection string, s func(collection *mongo.Collection) error) error {
	mongoDB, err := mongoSession()
	if err != nil {
		return err
	}
	c := mongoDB.Database(database).Collection(collection)
	return s(c)
}
