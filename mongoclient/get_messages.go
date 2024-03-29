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

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
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

type QueryParameters struct {
	Collection string
	Topic      string
	From       int64
	To         int64
	Limit      int64
	PlayerID   string
	IsBlocked  bool
}

// GetMessages returns messages stored in MongoDB by topic
// since MongoDB uses the MessageV2 format, this method converts
// the MessageV2 model into the Message one for retrocompatibility
// Rhe main difference being that the payload field is now referred to as "original_payload" and
// is a JSON object, not a string, and also the timestamp is int64 seconds since Unix epoch, not an ISODate
func GetMessages(ctx context.Context, queryParameters QueryParameters) []*models.Message {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_messages")
	defer span.Finish()
	searchResults := GetMessagesV2(ctx, queryParameters)
	messages := make([]*models.Message, 0)
	for _, result := range searchResults {
		messages = append(messages, ConvertMessageV2ToMessage(result))
	}

	return messages
}

func ConvertMessageV2ToMessage(messagev2 *models.MessageV2) *models.Message {
	pBytes, _ := json.Marshal(messagev2.Payload)
	return &models.Message{
		Timestamp: time.Unix(messagev2.Timestamp, 0),
		Payload:   string(pBytes),
		Topic:     messagev2.Topic,
	}
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

func GetMessagesPlayerSupportV2WithParameter(ctx context.Context, queryParameters QueryParameters) []*models.MessageV2 {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_messages_player_support_v2_with_parameter")
	defer span.Finish()

	mongoCollection, err := GetCollection(ctx, queryParameters.Collection)
	if err != nil {
		span.SetTag("collection", queryParameters.Collection)
		ext.LogError(span, err, log.Message("Error getting collection from MongoDB"))
		logger.Logger.Warningf("Error getting collection from MongoDB: %s", err.Error())
		return []*models.MessageV2{}
	}

	rawResults, err := getMessagesPlayerSupportFromCollection(ctx, queryParameters, mongoCollection)
	if err != nil {
		ext.LogError(span, err, log.Message("Error getting messages from MongoDB"))
		logger.Logger.Warningf("Error getting messages from MongoDB: %s", err.Error())
		return []*models.MessageV2{}
	}

	// convert the raw results to the MessageV2 model
	searchResults := make([]*models.MessageV2, len(rawResults))
	for i := 0; i < len(rawResults); i++ {
		searchResults[i], err = convertRawMessageToModelMessage(rawResults[i])

		if err != nil {
			ext.LogError(span, err, log.Message("Error converting messages from MongoDB"))
			logger.Logger.Warningf("Error converting messages from MongoDB: %s", err.Error())
			return []*models.MessageV2{}
		}
	}

	return searchResults
}

func getMessagesPlayerSupportFromCollection(
	ctx context.Context,
	queryParameters QueryParameters,
	mongoCollection *mongo.Collection,
) ([]MongoMessage, error) {
	query := resolveQuery(queryParameters)
	sort := bson.D{
		{"topic", 1},
		{"timestamp", -1},
	}

	statement := ExtractStatementForTrace(query, sort, queryParameters.Limit)
	span, ctx := opentracing.StartSpanFromContext(
		ctx,
		"get_messages_player_support_from_collection",
		opentracing.Tags{
			string(ext.DBStatement): statement,
			string(ext.DBType):      "mongo",
			string(ext.DBInstance):  mongoCollection.Database().Name(),
			string(ext.DBUser):      user,
			"collection":            mongoCollection.Name(),
		},
	)
	defer span.Finish()

	opts := options.Find()
	opts.SetSort(sort)
	opts.SetLimit(queryParameters.Limit)

	cursor, err := mongoCollection.Find(ctx, query, opts)
	if err != nil {
		ext.LogError(span, err, log.Message("Error finding messages in MongoDB"))
		return nil, err
	}

	rawResults := make([]MongoMessage, 0)
	if err = cursor.All(ctx, &rawResults); err != nil {
		ext.LogError(span, err, log.Message("Error decoding messages of a cursor from MongoDB"))
		return nil, err
	}

	return rawResults, nil
}

func resolveQuery(queryParameters QueryParameters) bson.M {
	query := bson.M{
		"timestamp": bson.M{
			"$gte": queryParameters.From,
			"$lte": queryParameters.To,
		},
		"blocked": queryParameters.IsBlocked,
	}

	if queryParameters.Topic != "" {
		query["topic"] = queryParameters.Topic
	}

	if queryParameters.PlayerID != "" {
		query["player_id"] = queryParameters.PlayerID
	}

	return query
}

func ExtractStatementForTrace(query bson.M, sort bson.D, limit int64) string {
	queryCopy := make(map[string]interface{}, len(query))
	for k, v := range query {
		queryCopy[k] = v
	}
	queryCopy["sort"] = sort
	queryCopy["limit"] = limit
	statementByteArray, _ := bson.MarshalExtJSON(queryCopy, true, true)
	return string(statementByteArray)
}
