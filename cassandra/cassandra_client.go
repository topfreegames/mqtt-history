package cassandra

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"github.com/topfreegames/extensions/cassandra"
	"github.com/topfreegames/extensions/middleware"
	"github.com/topfreegames/mqtt-history/models"

	cassandrainterfaces "github.com/topfreegames/extensions/cassandra/interfaces"
)

func sendMetrics(ctx context.Context, mr middleware.MetricsReporter, keyspace string, elapsed time.Duration, logger *logging.Logger) {
	logger.Debug("[sendMetrics] sending metrics do statsd")

	if mr == nil || ctx == nil {
		if mr == nil {
			logger.Debug("MetricsReporter is nil")
		} else {
			logger.Debug("ctx is nil")
		}
		return
	}

	tags := []string{fmt.Sprintf("keyspace:%s", keyspace)}

	if val, ok := ctx.Value("queryName").(string); ok {
		tags = append(tags, fmt.Sprintf("queryName:%s", val))
	}

	logger.Debug("sending metrics to statsd")

	if err := mr.Timing("cassandraQuery", elapsed, tags...); err != nil {
		logger.Errorf("[sendMetrics] failed to send metric to statsd: %s", err.Error())
	}
}

// QueryObserver implements gocql.QueryObserver
type QueryObserver struct {
	logger          *logging.Logger
	MetricsReporter middleware.MetricsReporter
}

// ObserveQuery sends timing metrics to dogstatsd on every query
func (o *QueryObserver) ObserveQuery(ctx context.Context, q gocql.ObservedQuery) {
	sendMetrics(ctx, o.MetricsReporter, q.Keyspace, q.End.Sub(q.Start), o.logger)
}

// BatchObserver implements gocql.BatchObserver
type BatchObserver struct {
	logger          *logging.Logger
	MetricsReporter middleware.MetricsReporter
}

// ObserveBatch sends timing metrics to dogstatsd on every batch query
func (o *BatchObserver) ObserveBatch(ctx context.Context, b gocql.ObservedBatch) {
	sendMetrics(ctx, o.MetricsReporter, b.Keyspace, b.End.Sub(b.Start), o.logger)
}

// Store is the access layer and contains the cassandra session.
// Implements DataStore
type Store struct {
	DBSession cassandrainterfaces.Session
	bucket    *models.Bucket
	logger    *logging.Logger
}

// GetCassandra connects on Cassandra and returns the client with a session
func GetCassandra(
	logger *logging.Logger,
	config *viper.Viper,
	mr middleware.MetricsReporter,
) (DataStore, error) {
	params := &cassandra.ClientParams{
		ClusterConfig: cassandra.ClusterConfig{
			Prefix:        "cassandra",
			QueryObserver: &QueryObserver{logger: logger, MetricsReporter: mr},
			BatchObserver: &BatchObserver{logger: logger, MetricsReporter: mr},
		},
		Config: config,
	}

	client, err := cassandra.NewClient(params)
	if err != nil {
		logger.Errorf("[GetCassandra] connection to database failed: %s", err.Error())
		return nil, err
	}

	logger.Info("successfully connected to cassandra")

	bucket := models.NewBucket(config)

	store := &Store{
		DBSession: client.Session,
		bucket:    bucket,
		logger:    logger,
	}

	return store, nil
}
