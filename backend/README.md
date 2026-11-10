# backend (Go)

REST API и WebSocket-сервер. Текущее состояние — этап M1: каркас, БД и health.

- `cmd/api/main.go` — точка входа, graceful shutdown.
- `internal/config` — чтение env и значения по умолчанию.
- `internal/transport/http` — Gin-роутер, middleware, хендлеры.
- `internal/model` — доменные структуры.
- `internal/repository/postgres` — пул pgx и SQL-запросы.
- `internal/service`, `internal/auth`, `internal/ws`,
  `internal/integration/cv` — заготовки следующих этапов.
- `migrations` — SQL-миграции golang-migrate.
- `api/openapi.yaml` — контракт API (появится позже).

## Команды

Из корня репозитория:

```bash
make up            # postgres + миграции + backend в Docker
make ps            # статус контейнеров
make logs          # логи
make down          # остановить
make migrate-up    # применить миграции
make migrate-down  # откатить последнюю
make psql          # psql внутри контейнера
```

Локальный запуск (нужен .env в корне и Postgres на 5432):

```bash
cd backend && go run ./cmd/api
```

Проверки:

```bash
go build ./... && go vet ./... && go test ./...
```

## Эндпоинты M1

- `GET /api/health` — `200 {"status":"ok","db":"ok"}` или
  `503 {"status":"degraded","db":"down"}`.
- `GET /api/parking-lots` — список парковок (пока пустой массив).

Ошибки отдаются в формате:

```json
{ "error": { "code": "not_found", "message": "route not found" } }
```

## Порты

- Backend на хосте — `8080` (`HTTP_PORT`), внутри контейнера — `8080`.
- Postgres на хосте — `5432` (`POSTGRES_PORT`), внутри — `5432`.
