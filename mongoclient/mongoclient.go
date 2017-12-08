// mqtt-history
// https://github.com/topfreegames/mqtt-history
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2017 Top Free Games <backend@tfgco.com>

package mongoclient

import (
	"github.com/spf13/viper"
	"gopkg.in/mgo.v2"
)

var (
	client *MongoSession
)

//MongoSession is a mongo session
type MongoSession struct {
	Session *mgo.Session
}

//GetMongoSession returns a MongoSession
func GetMongoSession() *MongoSession {
	client = &MongoSession{}
	if client.Session == nil {
		var err error
		client.Session, err = mgo.Dial(viper.GetString("mongo.host"))
		if err != nil {
			panic(err)
		}
	}
	return client
}

//GetCollection returns a collection from the database
func GetCollection(database string, collection string, s func(*mgo.Collection) error) error {
	Session := GetMongoSession().Session.Clone()
	defer Session.Close()
	c := Session.DB(database).C(collection)
	return s(c)
}
