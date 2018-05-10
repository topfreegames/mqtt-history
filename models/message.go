package models

import "time"

// Message represents a chat message
type Message struct {
	Timestamp time.Time `json:"timestamp"`
	Payload   string    `json:"payload"`
	Topic     string    `json:"topic"`
}
