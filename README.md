# MqttHistory

[![Build Status](https://travis-ci.org/topfreegames/mqtt-history.svg?branch=master)](https://travis-ci.org/topfreegames/mqtt-history)
[![Coverage Status](https://coveralls.io/repos/github/topfreegames/mqtt-history/badge.svg?branch=master)](https://coveralls.io/github/topfreegames/mqtt-history?branch=master)

An MQTT-based history handler for messages recorded by [mqttbot](https://github.com/topfreegames/mqttbot) in Cassandra

There's also support for messages stored in MongoDB, assuming the message documents contain these **required** fields:
```
{
    "topic": "<mqtt history target topic name>",
    "original_payload": "<message payload>",
    "timestamp": <int64 seconds from Unix epoch>
}
```

## Features
- Listen to healthcheck requests
- Retrieve message history from Cassandra when requested by users
- Authorization handling with support for MongoDB or an HTTP Authorization API

## Setup

Make sure you have Go installed on your machine.

You also need to have access to running instances of Cassandra and Mongo.

## Running the application

If you want to run the application locally you can do so by running

```
make deps
make create-cassandra-table
make run
```

You may need to change the configurations to point to your MQTT, Cassandra
and Mongo servers, or you can use the provided containers, they can be run
by executing `make run-containers`

## Running the tests

The project is integrated with Travis CI and uses docker to run the needed services.

If you are interested in running the tests yourself you will need docker (version 1.10
and up) and docker-compose.

To run the tests simply run `make test`

## Authorization

The project supports checking whether a user is authorized to retrieve the message history for a given topic.
This can be done via either MongoDB- or HTTP-based authorization, depending on the configuration.

For MongoDB, which is the default method, the required settings are
```
mongo:
  host: "mongodb://localhost:27017"
  allow_anonymous: false # whether to make authorization checks or not
  database: "mqtt"
```

For HTTP auth, the required settings are
```
httpAuth:
  enabled: true # whether to use HTTP or MongoDB for authorization
  requestURL: "http://localhost:8080/auth" # endpoint to make auth requests
  timeout: 10 # request timeout in seconds
  iam:
    enabled: true # whether to use Basic Auth when accessing the Auth API
    credentials: # credentials for Basic Auth
      username: user
      password: pass
```
