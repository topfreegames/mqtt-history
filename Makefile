build:
	@go build -mod vendor -a -installsuffix cgo -o . .

vendor:
	@go mod vendor

tidy:
	@go mod tidy

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
	@go test -coverprofile=coverage.out -covermode=count ./...

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
