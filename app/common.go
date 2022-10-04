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
) ([]*models.Message, error) {
	messages := []*models.Message{}
	var err error

	for i := 0; i < bucketQuantity && len(messages) < limit; i++ {
		bucket := currentBucket - i
		if bucket < 0 {
			break
		}

		queryLimit := limit - len(messages)
		bucketMessages, er := cassandra.SelectMessagesInBucket(ctx, topic, bucket, queryLimit)
		err = er
		messages = append(messages, bucketMessages...)
	}

	return messages, err
}
