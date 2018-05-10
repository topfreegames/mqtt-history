package cassandra

import (
	"context"
	"fmt"
	"time"

	"github.com/topfreegames/mqtt-history/models"
)

// DataStore is the interface with data access methods
type DataStore interface {
	SelectMessagesInBucket(
		ctx context.Context, topic string,
		from int64,
		qnt, limit int,
	) []*models.Message

	SelectMessagesBeforeTime(
		ctx context.Context,
		topic string,
		from, to int64,
		limit int,
	) []*models.Message

	InsertWithTTL(
		ctx context.Context,
		topic, payload string,
		now time.Time,
		ttl ...time.Duration,
	) error
}

func (s *Store) exec(ctx context.Context, query string, params ...interface{}) []*models.Message {
	messages := []*models.Message{}
	iter := s.DBSession.Query(query, params...).WithContext(ctx).Iter()
	defer func() {
		err := iter.Close()
		if err != nil {
			s.logger.Errorf("failed to execute query: %+v", map[string]string{
				"query": query,
				"error": err.Error(),
			})
		}
	}()

	for {
		var payload, topic string
		var timestamp time.Time
		if !iter.Scan(&payload, &timestamp, &topic) {
			break
		}
		messages = append(messages, &models.Message{
			Timestamp: timestamp,
			Payload:   payload,
			Topic:     topic,
		})
	}

	return messages
}

// SelectMessagesInBucket gets at most limit messages on
// topic and bucket from Cassandra.
func (s *Store) SelectMessagesInBucket(
	ctx context.Context,
	topic string,
	from int64,
	qnt, limit int,
) []*models.Message {
	query := fmt.Sprintf(`
	SELECT payload, toTimestamp(id) as timestamp, topic
	FROM messages 
	WHERE topic = ? AND bucket IN ?
	LIMIT %d
	`, limit)

	buckets := s.bucket.GetBuckets(from, qnt)

	return s.exec(ctx, query, topic, buckets)
}

// SelectMessagesBeforeTime ...
func (s *Store) SelectMessagesBeforeTime(
	ctx context.Context,
	topic string,
	from, to int64,
	limit int,
) []*models.Message {
	query := fmt.Sprintf(`
	SELECT payload, toTimestamp(id) as timestamp, topic
	FROM messages 
	WHERE 
		topic = ? AND bucket IN ? 
	LIMIT %d
	`, limit)

	buckets := s.bucket.Range(from, to)

	return s.exec(ctx, query, topic, buckets)
}

// InsertWithTTL ...
func (s *Store) InsertWithTTL(
	ctx context.Context,
	topic, payload string,
	now time.Time,
	ttl ...time.Duration,
) error {
	ttlVar := 1 * time.Minute
	if len(ttl) > 0 {
		ttlVar = ttl[0]
	}

	query := fmt.Sprintf(`
	INSERT INTO messages(id, topic, payload, bucket)
	VALUES(now(), ?, ?, ?)
	USING TTL %d
	`, int(ttlVar.Seconds()))

	timestamp := now.Unix()
	err := s.DBSession.Query(query, topic, payload, s.bucket.Get(timestamp)).Exec()
	return err
}
