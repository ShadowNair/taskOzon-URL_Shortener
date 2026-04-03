APP=shortener

.PHONY: test lint build run-memory run-postgres docker-memory docker-postgres docker-db-up docker-db-down test-repo-postgres

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

docker-down:
	docker compose down -v

docker-db-up:
	docker compose --profile postgres up -d postgres

docker-db-down:
	docker compose --profile postgres down -v

test-repo-postgres:
	set -a; [ -f .env ] && . ./.env; set +a; RUN_PG_TESTS=1 go test -v ./internal/repository/postgres/link