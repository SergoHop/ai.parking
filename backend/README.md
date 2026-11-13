# backend (Go)

REST API и WebSocket-сервер. Текущее состояние — этап M2: авторизация админа,
CRUD парковок/мест/камер и seed демо-парковки.

- `cmd/api/main.go` — точка входа, graceful shutdown.
- `cmd/seed/main.go` — загрузка демо-парковки из JSON.
- `internal/config` — чтение env и значения по умолчанию.
- `internal/transport/http` — Gin-роутер, middleware, хендлеры.
- `internal/auth` — bcrypt, JWT и создание админа из env.
- `internal/service` — валидация входных данных.
- `internal/model` — доменные структуры и DTO.
- `internal/repository/postgres` — пул pgx и SQL-запросы.
- `internal/ws`, `internal/integration/cv` — заготовки следующих этапов.
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
make seed          # загрузить демо-парковку из data/lots/demo.json
make seed-reset    # пересоздать демо-парковку
```

Локальный запуск (нужен .env в корне и Postgres на 5432):

```bash
cd backend && go run ./cmd/api
```

Проверки:

```bash
go build ./... && go vet ./... && go test ./...
```

## Авторизация

Первый админ создаётся автоматически при старте, если его ещё нет:

```
ADMIN_USERNAME=admin
ADMIN_PASSWORD=...
```

Пароль хранится в БД только как bcrypt-хеш. Если изменить `ADMIN_PASSWORD`
и перезапустить backend, хеш обновится. Логин:

```bash
curl -X POST localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"..."}'
```

Ответ содержит `token`; дальше он передаётся как `Authorization: Bearer <token>`.

## Эндпоинты

Публичное чтение:

- `GET /api/health` — `200 {"status":"ok","db":"ok"}` или
  `503 {"status":"degraded","db":"down"}`.
- `GET /api/parking-lots` — список парковок со счётчиками мест.
- `GET /api/parking-lots/{id}` — парковка, места, камеры и статусы.
- `GET /api/parking-lots/{id}/spots` — места парковки.
- `GET /api/parking-lots/{id}/cameras` — камеры парковки.

Только админ (JWT):

- `POST /api/auth/login`
- `POST/PUT/DELETE /api/parking-lots[/{id}]`
- `POST /api/parking-lots/{id}/spots`
- `PUT/DELETE /api/spots/{id}`
- `POST/PUT/DELETE /api/cameras[/{id}]`

`PUT` — полная замена сущности. Ошибки отдаются в формате:

```json
{ "error": { "code": "not_found", "message": "route not found" } }
```

## Демо-данные

`data/lots/demo.json` содержит одну парковку, камеру и 8 мест. Полигоны
и координаты — временные, пока не выбрана реальная площадка. После выбора
меняются координаты и полигоны в JSON, затем:

```bash
make seed-reset
```

## Порты

- Backend на хосте — `8080` (`HTTP_PORT`), внутри контейнера — `8080`.
- Postgres на хосте — `5432` (`POSTGRES_PORT`), внутри — `5432`.
