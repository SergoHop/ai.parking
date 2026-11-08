# backend (Go)

REST API и WebSocket-сервер.

- `cmd/api/main.go` — точка входа.
- `internal/transport/http` — роутер и хендлеры.
- `internal/service` — бизнес-логика.
- `internal/auth` — (опционально) пользователи и роли.
- `internal/repository/postgres` — доступ к БД.
- `internal/integration/cv` — клиент Python-сервиса.
- `internal/ws` — рассылка live-обновлений статусов мест.
- `migrations` — SQL-миграции.
- `api/openapi.yaml` — контракт API.

Планируемые эндпоинты: `POST /api/videos` (загрузка файла),
`GET /api/videos/{id}` (статус обработки), `/api/parking-lots`,
`/api/parking-lots/{id}/spots`, `/api/spots/{id}/history`, `/api/stats`,
`/ws/live`.
