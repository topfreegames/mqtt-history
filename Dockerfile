FROM golang:1.8-alpine

MAINTAINER TFG Co <backend@tfgco.com>

RUN apk add --no-cache git bash

RUN go get -u github.com/Masterminds/glide/...

ADD . /go/src/github.com/topfreegames/mqtt-history

WORKDIR /go/src/github.com/topfreegames/mqtt-history
RUN glide install
RUN go install github.com/topfreegames/mqtt-history

ENV MQTTHISTORY_ELASTICSEARCH_HOST http://localhost:9200
ENV MQTTHISTORY_ELASTICSEARCH_SNIFF false

ENV MQTTHISTORY_REDIS_HOST localhost
ENV MQTTHISTORY_REDIS_PORT 6379
ENV MQTTHISTORY_API_TLS false
ENV MQTTHISTORY_API_CERTFILE ./misc/example.crt
ENV MQTTHISTORY_API_KEYFILE ./misc/example.key
ENV MQTTHISTORY_CONFIG_FILE ./config/local.yml

EXPOSE 5000

CMD ./start_docker.sh
