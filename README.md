# MqttHistory

[![Build Status](https://travis-ci.org/topfreegames/mqtt-history.svg?branch=master)](https://travis-ci.org/topfreegames/mqtt-history)
[![Coverage Status](https://coveralls.io/repos/github/topfreegames/mqtt-history/badge.svg?branch=master)](https://coveralls.io/github/topfreegames/mqtt-history?branch=master)

An MQTT-based history handler for messages recorded by [mqttbot](https://github.com/topfreegames/mqttbot) in Cassandra


## Features

MqttHistory is an extensible

The bot is capable of:
- Listen to healthcheck requests
- Send history messages requested by users

## Setup

Make sure you have go installed on your machine.

You also need to have access to running instances of Cassandra and Mongo.

## Running the application

If you want to run the application locally you can do so by running

```
make setup
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
