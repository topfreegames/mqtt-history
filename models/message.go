package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Message represents a chat message
type Message struct {
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Payload   string    `json:"payload" bson:"payload"`
	Topic     string    `json:"topic" bson:"topic"`
}

// MongoMessage represents a chat message stored in Mongo.
type MongoMessage struct {
	Timestamp int64 `bson:"timestamp"`
	Payload bson.M `bson:"original_payload"`
	Topic string `bson:"topic"`
}
