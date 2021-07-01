package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/topfreegames/mqtt-history/cassandra"
	"github.com/topfreegames/mqtt-history/models"
	"github.com/topfreegames/mqtt-history/mongoclient"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func selectFromBuckets(
	ctx context.Context,
	bucketQuantity, limit, currentBucket int,
	topic string,
	cassandra cassandra.DataStore,
) []*models.Message {
	messages := []*models.Message{}

	for i := 0; i < bucketQuantity && len(messages) < limit; i++ {
		bucket := currentBucket - i
		if bucket < 0 {
			break
		}

		queryLimit := limit - len(messages)
		bucketMessages := cassandra.SelectMessagesInBucket(ctx, topic, bucket, queryLimit)
		messages = append(messages, bucketMessages...)
	}

	return messages
}

// SelectFromCollection expects the message to be stored in mongo with a specific structure,
// the main difference being that the payload field is now referred to as "original_payload" and
// is a JSON object, not a string, and also the timestamp is int64 seconds since Unix epoch, not an ISODate on Mongo
func SelectFromCollection(ctx context.Context, topic string, from int64, limit int64, collection string) []*models.Message {
	searchResults := make([]models.MongoMessage, 0)

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

	err := mongoclient.GetCollection(collection, callback)
	if err != nil {
		return []*models.Message{}
	}

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
