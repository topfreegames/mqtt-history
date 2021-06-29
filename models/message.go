package models

import "time"

// Message represents a chat message
type Message struct {
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Payload   string    `json:"payload" bson:"payload"`
	Topic     string    `json:"topic" bson:"topic"`
}
