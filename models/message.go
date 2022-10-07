package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Message represents a chat message stored in MongoDB
type Message struct {
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Payload   string    `json:"payload" bson:"payload"`
	Topic     string    `json:"topic" bson:"topic"`
}

// MessageV2 represents new payload for the chat message
// This payload is stored in MongoDB
// v2 endpoints return this format,
// while v1 endpoints convert this format to the originalMessage model
type MessageV2 struct {
	Id             string `json:"id" bson:"id"`
	Timestamp      int64  `json:"timestamp" bson:"timestamp"`
	Payload        bson.M `json:"original_payload" bson:"original_payload"`
	Topic          string `json:"topic" bson:"topic"`
	PlayerId       string `json:"player_id" bson:"player_id"`
	Message        string `json:"message" bson:"message"`
	GameId         string `json:"game_id" bson:"game_id"`
	Blocked        bool   `json:"blocked" bson:"blocked"`
	ShouldModerate bool   `json:"should_moderate" bson:"should_moderate"`
	Metadata       bson.M `json:"metadata" bson:"metadata"`
}
