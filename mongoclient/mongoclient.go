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

	"github.com/spf13/viper"
	"github.com/topfreegames/extensions/mongo"
	"github.com/topfreegames/extensions/mongo/interfaces"
)

var (
	client *mongo.Client
	once   sync.Once
)

//GetMongoSession returns a MongoSession
func GetMongoSession() (interfaces.MongoDB, error) {
	var err error

	once.Do(func() {
		config := viper.GetViper()

		url := config.Get("mongo.host")
		config.Set("mongo.url", url)

		client, err = mongo.NewClient("mongo", config)
	})

	if err != nil {
		return nil, err
	}
	return client.MongoDB, nil
}

//GetCollection returns a collection from the database
func GetCollection(ctx context.Context, collection string, s func(interfaces.Collection) error) error {
	mongoDB, err := GetMongoSession()
	if err != nil {
		return err
	}
	c, session := mongoDB.WithContext(ctx).C(collection)
	defer session.Close()
	return s(c)
}
