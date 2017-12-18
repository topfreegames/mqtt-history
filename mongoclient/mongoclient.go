// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2017 Top Free Games <backend@tfgco.com>

package mongoclient

import (
	"sync"

	"github.com/spf13/viper"
	mgo "gopkg.in/mgo.v2"
)

var (
	client *MongoSession
	once   sync.Once
)

//MongoSession is a mongo session
type MongoSession struct {
	Session *mgo.Session
}

//GetMongoSession returns a MongoSession
func GetMongoSession() (*MongoSession, error) {
	once.Do(func() {
		client = &MongoSession{}
	})
	if client.Session == nil {
		var err error
		client.Session, err = mgo.Dial(viper.GetString("mongo.host"))
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}

//GetCollection returns a collection from the database
func GetCollection(database string, collection string, s func(*mgo.Collection) error) error {
	mongoSession, err := GetMongoSession()
	if err != nil {
		return err
	}
	session := mongoSession.Session.Clone()
	defer session.Close()
	c := session.DB(database).C(collection)
	return s(c)
}
