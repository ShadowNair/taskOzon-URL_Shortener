APP=shortener

.PHONY: test lint build run-memory run-postgres docker-memory docker-postgres

test:
	go test -cover ./...

lint:
	golangci-lint run ./...

build:
	go build -o bin/$(APP) ./cmd/app

run-memory:
	go run ./cmd/app -storage=memory

run-postgres:
	STORAGE_TYPE=postgres go run ./cmd/app -storage=postgres

docker-memory:
	docker compose --profile memory up --build -d

docker-postgres:
	docker compose --profile postgres up --build -d