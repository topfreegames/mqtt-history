package models

// Defaults saves the default configs
type Defaults struct {
	LimitOfMessages         int64
	MongoEnabled            bool
	MongoMessagesCollection string
}
