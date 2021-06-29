package models

// Defaults saves the default configs
type Defaults struct {
	BucketQuantityOnSelect  int
	LimitOfMessages         int64
	MongoEnabled            bool
	MongoMessagesCollection string
}
