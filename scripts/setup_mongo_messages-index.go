package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

const (
	addressEnvVar    = "MONGO_URL"
	databaseEnvVar   = "MONGO_DATABASE"
	collectionEnvVar = "MONGO_COLLECTION"

	TTL = 6 * 31 * 24 * time.Hour // 6 months
)

var (
	ascending = bsonx.Int32(1)
	descending = bsonx.Int32(-1)
)

func main() {
	address := getConfig(addressEnvVar, "mongodb://localhost:27017")
	database := getConfig(databaseEnvVar, "chat")
	collection := getConfig(collectionEnvVar, "messages")

	const defaultTimeout = 10
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(address))
	if err != nil {
		panic(err)
	}

	db := client.Database(database)
	coll := db.Collection(collection)

	err = createTopicIndex(coll)
	if err != nil {
		panic(err)
	}
	fmt.Println("Created 'topic' index")

	err = createUserIndex(coll)
	if err != nil {
		panic(err)
	}
	fmt.Println("Created 'user_id' index")

	err = createTTLIndex(coll)
	if err != nil {
		panic(err)
	}
	fmt.Println("Created 'TTL' index")
}

func getConfig(envVar, fallback string) string {
	value := os.Getenv(envVar)
	if len(value) == 0 {
		fmt.Printf("%s env var not set\n", envVar)
		fmt.Printf("Using '%s' as default value\n", fallback)
		value = fallback
	}
	return value
}

func createTopicIndex(coll *mongo.Collection) error {
	opts := options.Index()
	opts.SetName("topic_timestamp")

	index := mongo.IndexModel{
		Keys: bsonx.Doc{
			{
				Key:   "topic",
				Value: ascending,
			},
			{
				Key:   "timestamp",
				Value: descending,
			},
		},
		Options: opts,
	}

	return createIndex(index, coll)
}

func createUserIndex(coll *mongo.Collection) error {
	opts := options.Index()
	opts.SetName("user_timestamp")

	index := mongo.IndexModel{
		Keys: bsonx.Doc{
			{
				Key:   "user_id",
				Value: ascending,
			},
			{
				Key:   "timestamp",
				Value: descending,
			},
		},
		Options: opts,
	}

	return createIndex(index, coll)
}

func createTTLIndex(coll *mongo.Collection) error {
	opts := options.Index()
	opts.SetExpireAfterSeconds(int32(TTL / time.Second))
	opts.SetName("messages_TTL")

	index := mongo.IndexModel{
		Keys: bsonx.Doc{
			{
				Key:   "timestamp",
				Value: descending,
			},
		},
		Options: opts,
	}

	return createIndex(index, coll)
}

func createIndex(index mongo.IndexModel, coll *mongo.Collection) error {
	indexes := coll.Indexes()

	opts := options.CreateIndexes()
	_, err := indexes.CreateOne(context.Background(), index, opts)
	if err != nil {
		return err
	}
	return nil
}
