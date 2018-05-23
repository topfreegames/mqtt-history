package app

import (
	"context"

	"github.com/topfreegames/mqtt-history/cassandra"
	"github.com/topfreegames/mqtt-history/models"
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
