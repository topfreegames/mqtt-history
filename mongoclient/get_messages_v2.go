package mongoclient

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/topfreegames/mqtt-history/logger"
	"github.com/topfreegames/mqtt-history/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetMessagesV2 returns messages stored in MongoDB by topic
// It returns the MessageV2 model that is stored in MongoDB
func GetMessagesV2(ctx context.Context, queryParameters QueryParameters) []*models.MessageV2 {
	return GetMessagesV2WithParameter(ctx, queryParameters)
}

func GetMessagesV2WithParameter(ctx context.Context, queryParameters QueryParameters) []*models.MessageV2 {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_messages_v2_with_parameter")
	defer span.Finish()

	mongoCollection, err := GetCollection(ctx, queryParameters.Collection)
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(
			log.Event("error"),
			log.Message("Error getting collection from MongoDB"),
			log.Error(err),
		)
		logger.Logger.Warning("Error getting collection from MongoDB", err)
		return []*models.MessageV2{}
	}

	rawResults, err := getMessagesFromCollection(ctx, queryParameters, mongoCollection)
	if err != nil {
		logger.Logger.Warning("Error getting messages from MongoDB", err)
		return []*models.MessageV2{}
	}

	// convert the raw results to the MessageV2 model
	searchResults := make([]*models.MessageV2, len(rawResults))
	for i := 0; i < len(rawResults); i++ {
		searchResults[i], err = convertRawMessageToModelMessage(rawResults[i])

		if err != nil {
			span.SetTag("error", true)
			span.LogFields(
				log.Event("error"),
				log.Message("Error converting messages from MongoDB to the program domain format"),
				log.Error(err),
			)
			logger.Logger.Warningf("Error converting messages from MongoDB: %s", err.Error())
			return []*models.MessageV2{}
		}
	}

	return searchResults
}

func getMessagesFromCollection(
	ctx context.Context,
	queryParameters QueryParameters,
	mongoCollection *mongo.Collection,
) ([]MongoMessage, error) {
	query := bson.M{
		"topic": queryParameters.Topic,
		"timestamp": bson.M{
			"$lte": queryParameters.From,
		},
		"blocked": queryParameters.IsBlocked,
	}
	sort := bson.D{
		{"topic", 1},
		{"timestamp", -1},
	}

	statement := extractStatementForTrace(query, sort, queryParameters.Limit)
	span, ctx := opentracing.StartSpanFromContext(
		ctx,
		"get_messages_from_collection",
		opentracing.Tags{
			string(ext.DBStatement): statement,
			string(ext.DBType):      "mongo",
			string(ext.DBInstance):  database,
			string(ext.DBUser):      user,
		},
	)
	defer span.Finish()

	opts := options.Find()
	opts.SetSort(sort)
	opts.SetLimit(queryParameters.Limit)

	cursor, err := mongoCollection.Find(ctx, query, opts)
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(
			log.Event("error"),
			log.Message("Error finding messages in MongoDB"),
			log.Error(err),
		)
		return nil, err
	}

	rawResults := make([]MongoMessage, 0)
	if err = cursor.All(ctx, &rawResults); err != nil {
		span.SetTag("error", true)
		span.LogFields(
			log.Event("error"),
			log.Message("Error decoding messages of a cursor from MongoDB"),
			log.Error(err),
		)
		return nil, err
	}

	return rawResults, nil
}
