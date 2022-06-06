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
	"fmt"
	"time"

	"github.com/topfreegames/mqtt-history/logger"
	"github.com/topfreegames/mqtt-history/models"
	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoMessage represents new payload for the chat message
// that is stored in MongoDB
type MongoMessage struct {
	Id             string      `json:"id" bson:"id"`
	Timestamp      int64       `json:"timestamp" bson:"timestamp"`
	Payload        bson.M      `json:"original_payload" bson:"original_payload"`
	Topic          string      `json:"topic" bson:"topic"`
	PlayerId       interface{} `json:"player_id" bson:"player_id"`
	Message        string      `json:"message" bson:"message"`
	GameId         string      `json:"game_id" bson:"game_id"`
	Blocked        bool        `json:"blocked" bson:"blocked"`
	ShouldModerate bool        `json:"should_moderate" bson:"should_moderate"`
	Metadata       bson.M      `json:"metadata" bson:"metadata"`
}

// GetMessagesV2 returns messages stored in MongoDB by topic
// It returns the MessageV2 model that is stored in MongoDB

func GetMessagesV2(ctx context.Context, topic string, from int64, limit int64, collection string) []*models.MessageV2 {
	return GetMessagesV2WithParameter(ctx, topic, from, limit, collection, false)
}

func GetMessagesV2WithParameter(ctx context.Context, topic string, from int64, limit int64, collection string, isBlocked bool) []*models.MessageV2 {
	rawResults := make([]MongoMessage, 0)

	callback := func(coll *mongo.Collection) error {
		query := bson.M{
			"topic": topic,
			"timestamp": bson.M{
				"$lte": from, // less than or equal
			},
			"blocked": isBlocked,
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

		return cursor.All(ctx, &rawResults)
	}

	// retrieve the collection data
	err := GetCollection(collection, callback)
	if err != nil {
		logger.Logger.Warningf("Error getting messages from MongoDB: %s", err.Error())
		return []*models.MessageV2{}
	}

	// convert the raw results to the MessageV2 model
	searchResults := make([]*models.MessageV2, len(rawResults))

	for i := 0; i < len(rawResults); i++ {
		searchResults[i], err = convertRawMessageToModelMessage(rawResults[i])

		if err != nil {
			logger.Logger.Warningf("Error getting messages from MongoDB: %s", err.Error())
			return []*models.MessageV2{}
		}
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

func convertRawMessageToModelMessage(rawMessage MongoMessage) (*models.MessageV2, error) {
	playerIdAsString, err := convertPlayerIdToString(rawMessage.PlayerId)
	if err != nil {
		return nil, err
	}

	return &models.MessageV2{
		Id:             rawMessage.Id,
		Timestamp:      rawMessage.Timestamp,
		Payload:        rawMessage.Payload,
		Topic:          rawMessage.Topic,
		PlayerId:       playerIdAsString,
		Message:        rawMessage.Message,
		GameId:         rawMessage.GameId,
		Blocked:        rawMessage.Blocked,
		ShouldModerate: rawMessage.ShouldModerate,
		Metadata:       rawMessage.Metadata,
	}, nil
}

func convertPlayerIdToString(playerID interface{}) (string, error) {
	// TODO: refactor this code using switch to improve readability

	_, ok := playerID.(string)
	if ok {
		// force sprintf to avoid encoding issues
		return fmt.Sprintf("%s", playerID), nil
	}

	playerIdAsFloat32, ok := playerID.(float32)
	if ok {
		return fmt.Sprintf("%f0", playerIdAsFloat32), nil
	}

	playerIdAsFloat64, ok := playerID.(float64)
	if ok {
		return fmt.Sprintf("%f0", playerIdAsFloat64), nil
	}

	playerIdAsInt32, ok := playerID.(int32)
	if ok {
		return fmt.Sprintf("%d", playerIdAsInt32), nil
	}

	playerIdAsInt64, ok := playerID.(int64)
	if ok {
		return fmt.Sprintf("%d", playerIdAsInt64), nil
	}

	return "", fmt.Errorf("error converting player id to float64 or string. player id raw value: %s", playerID)
}

func GetMessagesPlayerSupportV2WithParameter(ctx context.Context, topic string, from int64, to int64, limit int64,
	collection string, isBlocked bool, playerId string) []*models.MessageV2 {
	rawResults := make([]MongoMessage, 0)

	callback := func(coll *mongo.Collection) error {
		query := bson.M{
			"timestamp": bson.M{
				"$gte": from, //greather than or equal
				"$lte": to,   // less than or equal
			},
			"player_id": playerId,
			"blocked":   isBlocked,
			"topic":     topic,
		}
		sort := bson.D{
			{"topic", 1},
			{"timestamp", -1},
		}

		if topic == "" {
			query = bson.M{
				"timestamp": bson.M{
					"$gte": from,
					"$lte": to,
				},
				"player_id": playerId,
				"blocked":   isBlocked,
			}
			sort = bson.D{
				{"topic", 1},
				{"timestamp", -1},
			}
		}

		if playerId == "" {
			query = bson.M{
				"timestamp": bson.M{
					"$gte": from,
					"$lte": to,
				},
				"topic":   topic,
				"blocked": isBlocked,
			}
			sort = bson.D{
				{"topic", 1},
				{"timestamp", -1},
			}
		}

		opts := options.Find()
		opts.SetSort(sort)
		opts.SetLimit(limit)

		cursor, err := coll.Find(ctx, query, opts)
		if err != nil {
			return err
		}

		return cursor.All(ctx, &rawResults)
	}
	// retrieve the collection data
	err := GetCollection(collection, callback)
	if err != nil {
		logger.Logger.Warningf("Error getting messages from MongoDB: %s", err.Error())
		return []*models.MessageV2{}
	}

	// convert the raw results to the MessageV2 model
	searchResults := make([]*models.MessageV2, len(rawResults))

	for i := 0; i < len(rawResults); i++ {
		searchResults[i], err = convertRawMessageToModelMessage(rawResults[i])

		if err != nil {
			logger.Logger.Warningf("Error getting messages from MongoDB: %s", err.Error())
			return []*models.MessageV2{}
		}
	}

	return searchResults
}
