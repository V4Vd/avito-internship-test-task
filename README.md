# PR Reviewer Assignment Service

Микросервис для автоматического назначения ревьюверов на PR (до 2 активных участников из команды автора).

## TL;DR

```bash
make deps && make docker-up
```

Сервис запустится на <http://localhost:8080>

## Тестирование

```bash
docker compose up -d postgres && go test -v ./...
```

## Линтинг

Проект использует `golangci-lint` для проверки качества кода:

Установка golangci-lint:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

```bash

# Проверить код
make lint

# Автоматически исправить проблемы
make lint-fix
```

Настройки линтера: [`.golangci.yml`](.golangci.yml)

## API примеры

Создать команду:

```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{"team_name":"backend","members":[{"user_id":"u1","username":"Alice","is_active":true}]}'
```

Создать PR (авто-назначение ревьюверов):

```bash
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1","pull_request_name":"Feature","author_id":"u1"}'
```

Merge PR:

```bash
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1"}'
```

Переназначить ревьювера:

```bash
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id":"pr-1","old_user_id":"u2"}'
```

Документация API:

- OpenAPI спецификация: [`openapi.yml`](openapi.yml)
- Postman коллекция: [`postman-collection.json`](postman-collection.json)

Импортируйте `postman-collection.json` в Postman для готовых примеров запросов.

## Переменные окружения

- `DATABASE_URL` PostgreSQL connection string

- `SERVER_ADDR` адрес сервера (по умолчанию `:8080`)
