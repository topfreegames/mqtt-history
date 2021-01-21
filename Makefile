PACKAGES= $(shell find . -type f -name "*.go" ! \( -path "*vendor*" \) | sed -En "s/([^\.])\/.*/\1/p" | uniq)
#PACKAGES = $(shell glide novendor)
GODIRS = $(shell go list ./... | grep -v /vendor/ | sed s@github.com/topfreegames/mqtt-history@.@g | egrep -v "^[.]$$")

setup:
	@go get -u github.com/golang/dep/...
	@dep ensure

setup-ci:
	@go get -u github.com/golang/dep/...
	@go get github.com/mattn/goveralls
	@go run github.com/golang/dep ensure

build:
	@go build $(PACKAGES)
	@go build

run-containers:
	@cd test_containers && docker-compose up -d && cd ..

kill-containers:
	@cd test_containers && docker-compose stop && cd ..

CASSANDRA_CONTAINER := mqtt-history_cassandra_1
create-cassandra-table:
	@until docker exec $(CASSANDRA_CONTAINER) cqlsh -e 'describe cluster'; do echo 'Waiting for Cassandra...' && sleep 2; done
	@echo 'Creating keyspace and table on Cassandra'
	@docker exec $(CASSANDRA_CONTAINER) cqlsh -e "$$(cat scripts/create.cql)";
	@echo 'Done'

run-tests: run-containers
	@make CASSANDRA_CONTAINER=mqtthistory_test_cassandra create-cassandra-table
	@make coverage
	@make kill-containers

test: run-tests

coverage:
	@echo "mode: count" > coverage-all.out
	@$(foreach pkg,$(PACKAGES),\
		echo "Testing $(pkg)" &&\
		go test -coverprofile=coverage.out -covermode=count $(pkg) || exit 1 &&\
		tail -n +2 coverage.out >> coverage-all.out;)

run:
	@go run main.go start

deps:
	@docker-compose up -d mongo cassandra

cross: cross-linux cross-darwin

cross-linux:
	@mkdir -p ./bin
	@echo "Building for linux-i386..."
	@env GOOS=linux GOARCH=386 go build -o ./bin/mqtt-history-linux-i386 ./main.go
	@echo "Building for linux-x86_64..."
	@env GOOS=linux GOARCH=amd64 go build -o ./bin/mqtt-history-linux-x86_64 ./main.go
	@$(MAKE) cross-exec

cross-darwin:
	@mkdir -p ./bin
	@echo "Building for darwin-i386..."
	@env GOOS=darwin GOARCH=386 go build -o ./bin/mqtt-history-darwin-i386 ./main.go
	@echo "Building for darwin-x86_64..."
	@env GOOS=darwin GOARCH=amd64 go build -o ./bin/mqtt-history-darwin-x86_64 ./main.go
	@$(MAKE) cross-exec

cross-exec:
	@chmod +x bin/*

default: @build
