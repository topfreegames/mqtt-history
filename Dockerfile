FROM golang:1.15-alpine AS build

MAINTAINER TFG Co <backend@tfgco.com>

WORKDIR /src

COPY vendor ./vendor

COPY . .

# Build a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -a -installsuffix cgo -o mqtt-history .

# Verify if the binary is truly static.
RUN ldd /src/mqtt-history 2>&1 | grep -q 'Not a valid dynamic program'

FROM alpine:3.13

COPY --from=build /src/mqtt-history ./mqtt-history
COPY --from=build /src/config ./config

ENV MQTTHISTORY_REDIS_HOST localhost
ENV MQTTHISTORY_REDIS_PORT 6379
ENV MQTTHISTORY_REDIS_DB 0
ENV MQTTHISTORY_API_TLS false
ENV MQTTHISTORY_API_CERTFILE ./misc/example.crt
ENV MQTTHISTORY_API_KEYFILE ./misc/example.key
ENV MQTTHISTORY_CONFIG_FILE ./config/local.yaml

EXPOSE 5000

CMD ./start_docker.sh
