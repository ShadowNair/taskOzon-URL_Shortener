# URL Shortener

Сервис сокращения ссылок на Go с двумя хранилищами:
- `memory`
- `postgres`

## API
- `POST /api/v1/links` - создать короткую ссылку
- `GET /{shortCode}` - получить оригинальный URL
- `GET /healthz` - healthcheck
- `GET /swagger/` - Swagger UI
- `GET /swagger/doc.json` - OpenAPI JSON

## Формат short code
- длина ровно 10 символов
- допустимые символы: `a-z`, `A-Z`, `0-9`, `_`

## Примеры запуска

### Локально, in-memory
```bash
make run-memory
```

### Локально, PostgreSQL
```bash
make run-postgres
```

### В Docker, in-memory
```bash
make docker-memory
```

### В Docker, PostgreSQL
```bash
make docker-postgres
```

### Опустить контейнеры
```bash
make docker-down
```

## Примеры запросов
```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}'
```

```bash
curl http://localhost:8080/abcDEF123_
```

## Тесты
```bash
make test
```

## Тесты для репозитория Postgres
```bash
make docker-db-up
make test-repo-postgres
make docker-db-down
```


## Линтер
```bash
make lint
```

## CI
GitHub Actions запускает:
- `go test ./...`
- `golangci-lint`
```

## `Dockerfile`

```dockerfile
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /shortener ./cmd/app

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /shortener /app/shortener

EXPOSE 8080
CMD ["/app/shortener"]
```