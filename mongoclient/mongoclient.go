// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2017 Top Free Games <backend@tfgco.com>

package mongoclient

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/topfreegames/mqtt-history/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"

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
	// staleness: check how old the data is before reading from Secondary replicas
	secondaryPreferredOpts := readpref.WithMaxStaleness(90 * time.Second)
	// secondaryPreferred: prefer reading from Secondary replicas, falling back to the primary if needed
	secondaryPreferred := readpref.SecondaryPreferred(secondaryPreferredOpts)
	dbOpts := options.Database().
		SetReadPreference(secondaryPreferred)

	c := mongoDB.Database(database, dbOpts).Collection(collection)
	return s(c)
}

// GetMessagesV2 returns messages stored in MongoDB by topic
// It returns the MessageV2 model that is stored in MongoDB
func GetMessagesV2(ctx context.Context, topic string, from int64, limit int64, collection string) []*models.MessageV2 {
	searchResults := make([]*models.MessageV2, 0)

	callback := func(coll *mongo.Collection) error {
		query := bson.M{
			"topic": topic,
			"timestamp": bson.M{
				"$lte": from, // less than or equal
			},
		}

		sort := bson.D{
			{"topic", 1},
			{"timestamp", -1},
		}

		opts := options.Find()
		opts.SetSort(sort)
		opts.SetLimit(limit)

		cursor, err := coll.Find(ctx, query, opts)
		if err != nil {
			return err
		}

		return cursor.All(ctx, &searchResults)
	}

	err := GetCollection(collection, callback)
	if err != nil {
		return []*models.MessageV2{}
	}

	return searchResults
}

// GetMessages returns messages stored in MongoDB by topic
// since MongoDB uses the MessageV2 format, this method converts
// the MessageV2 model into the Message one for retrocompatibility
// Rhe main difference being that the payload field is now referred to as "original_payload" and
// is a JSON object, not a string, and also the timestamp is int64 seconds since Unix epoch, not an ISODate
func GetMessages(ctx context.Context, topic string, from int64, limit int64, collection string) []*models.Message {
	searchResults := GetMessagesV2(ctx, topic, from, limit, collection)
	messages := make([]*models.Message, 0)
	for _, result := range searchResults {
		payload := result.Payload
		bytes, _ := json.Marshal(payload)

		finalStr := string(bytes)
		message := &models.Message{
			Timestamp: time.Unix(result.Timestamp, 0),
			Payload:   finalStr,
			Topic:     topic,
		}
		messages = append(messages, message)
	}

	return messages
}
