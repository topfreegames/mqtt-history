PROJECT_NAME := "mqtt-history"

help: Makefile ## show list of commands
	@echo "Choose a command run in "$(PROJECT_NAME)":"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort


build: ## build project
	@go build -mod vendor -a -installsuffix cgo -o . .

vendor:
	@go mod vendor

tidy:
	@go mod tidy

run-containers: ## run all test containers
	@cd test_containers && docker compose up -d && cd ..

kill-containers: ## kill all test containers
	@cd test_containers && docker compose down && cd ..

setup/mongo: 
	go run scripts/setup_mongo_messages-index.go

run-tests: run-containers ## run tests using the docker containers
	@make coverage
	@make kill-containers

test: run-tests ## run tests using the docker containers (alias to run-tests)

coverage:
	@go test -coverprofile=coverage.out -covermode=count ./...

run: ## start the API
	@go run main.go start

deps: ## start the API dependencies as docker containers
	@docker compose up -d mongo 

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
