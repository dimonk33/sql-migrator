.PHONY: build run test lint up down integration-tests

BIN_FILE := "./bin/migrator"
DOCKER_COMPOSE_PROD="./deployments/docker-compose.yaml"
DOCKER_COMPOSE_TEST="./deployments/docker-compose.test.yaml"

build:
	go build -v -o $(BIN_FILE) ./cmd/migrator

run: build
	$(BIN_FILE) -config ./configs/config.toml &&

test:
	go test -race ./internal/...

install-lint-deps:
	(which golangci-lint > /dev/null) \
	|| curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
	| sh -s -- -b $(shell go env GOPATH)/bin v1.52.2

lint: install-lint-deps
	golangci-lint run ./...

up:
	docker-compose -f $(DOCKER_COMPOSE_PROD) up --build -d ;

down:
	docker-compose -f $(DOCKER_COMPOSE_PROD) down ;

integration-tests:
	set -e ;\
	docker-compose -f $(DOCKER_COMPOSE_TEST) up --build -d ;\
	test_status_code=0 ;\
	docker-compose -f $(DOCKER_COMPOSE_TEST) run integration_tests go test -tags integration || test_status_code=$$? ;\
	docker-compose -f $(DOCKER_COMPOSE_TEST) down ;\
	exit $$test_status_code ;
