FROM golang:1.15-alpine AS build

MAINTAINER TFG Co <backend@tfgco.com>

WORKDIR /src

COPY vendor ./vendor

COPY . .

# Build a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -a -installsuffix cgo -o mqtt-history .

# Verify if the binary is truly static.
RUN ldd /src/mqtt-history 2>&1 | grep -q 'Not a valid dynamic program'

# build binary for migrations
RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -a -installsuffix cgo -o setup_mongo_messages-index ./scripts/setup_mongo_messages-index.go
RUN ldd /src/setup_mongo_messages-index 2>&1 | grep -q 'Not a valid dynamic program'

FROM alpine:3.13

COPY --from=build /src/mqtt-history ./mqtt-history
COPY --from=build /src/setup_mongo_messages-index ./setup_mongo_messages-index
COPY --from=build /src/config ./config

ENV MQTTHISTORY_API_TLS false
ENV MQTTHISTORY_API_CERTFILE ./misc/example.crt
ENV MQTTHISTORY_API_KEYFILE ./misc/example.key
ENV MQTTHISTORY_CONFIG_FILE ./config/local.yaml

EXPOSE 5000

CMD ./start_docker.sh
