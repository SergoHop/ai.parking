# API: REST и WebSocket

Бэкенд отдаёт JSON по `http://localhost:8080/api` и live-статусы по
`ws://localhost:8080/ws/live`. Все времена — RFC 3339 в UTC.

## Формат ошибок

```json
{ "error": { "code": "validation_error", "message": "invalid bbox bounds" } }
```

| Код | HTTP | Когда |
| --- | --- | --- |
| `validation_error` | 400 | Некорректное тело, параметры или `bbox` |
| `unauthorized` | 401 | Нет или просрочен Bearer-токен |
| `invalid_credentials` | 401 | Неверный логин/пароль |
| `forbidden` | 403 | Не хватает прав |
| `not_found` | 404 | Объект удалён или не существует |
| `conflict` | 409 | Нарушено уникальное ограничение |
| `invalid_reference` | 400 | Ссылка на чужую/несуществующую сущность |
| `internal_error` | 500 | Прочее |

## Свежесть статусов

Статус места в ответах — **эффективный**:

```
статус места, если камера места в статусе online
                 и её last_frame_at не старше FRESHNESS_TTL_SECONDS
unknown иначе
```

`FRESHNESS_TTL_SECONDS` по умолчанию `15`. В ответах есть признак `fresh`:
у парковки и места он означает «данные можно показывать как актуальные»,
у камеры — «online и кадр свежий». Агрегаты `free_spots`, `occupied_spots`,
`unknown_spots` пересчитываются по эффективным статусам, а не по сырым.

Публичный статус никогда не «додумывает» свободное место: если CV замолчал,
место становится `unknown`, а не `free`.

## Публичное чтение (без токена)

### `GET /api/health`

```json
{ "status": "ok", "db": "ok" }
```

Если БД недоступна — `503` и `{"status":"degraded","db":"down"}`.

### `GET /api/parking-lots`

Параметры:

| Параметр | Описание |
| --- | --- |
| `bbox` | `minLon,minLat,maxLon,maxLat` (порядок GeoJSON). Необязательный. |
| `include` | `spots` или `all` — вложить места в каждую парковку. |

Ответ — массив парковок:

```json
[
  {
    "id": 6,
    "name": "CV тест — видео парковки",
    "address": "Тестовая площадка (sample video)",
    "kind": "roadside",
    "center_lat": 55.75,
    "center_lon": 37.6,
    "radius_m": 100,
    "created_at": "2026-11-14T19:46:35Z",
    "total_spots": 7,
    "free_spots": 1,
    "occupied_spots": 6,
    "unknown_spots": 0,
    "last_frame_at": "2026-11-17T11:36:07Z",
    "fresh": true,
    "spots": []
  }
]
```

Поле `spots` присутствует только при `include=spots`.

### `GET /api/parking-lots/:id`

```json
{
  "lot": { "...": "поля как в списке, без spots" },
  "spots": [
    {
      "id": 22,
      "lot_id": 6,
      "camera_id": 5,
      "code": "T-04",
      "polygon": [[0.168, 0.648], [0.194, 0.648], [0.194, 0.785], [0.168, 0.785]],
      "map_lat": 55.75009,
      "map_lon": 37.60009,
      "status": "free",
      "confidence": 0.88,
      "status_updated_at": "2026-11-17T11:35:58Z",
      "fresh": true
    }
  ],
  "cameras": [
    {
      "id": 5,
      "lot_id": 6,
      "name": "Тестовое видео",
      "source_type": "file",
      "source": "data/videos/sample_parking_fixed.mp4",
      "status": "online",
      "last_frame_at": "2026-11-17T11:36:07Z",
      "fps": 12,
      "resolution": "1280x720",
      "fresh": true
    }
  ]
}
```

### `GET /api/parking-lots/:id/spots`

Массив мест, как в `spots` выше.

### `GET /api/parking-lots/:id/cameras`

Массив камер с признаком `fresh`.

## Авторизация

### `POST /api/auth/login`

```json
{ "username": "admin", "password": "..." }
```

```json
{
  "token": "eyJhbGciOi...",
  "expires_at": "2026-11-18T11:36:07Z",
  "user": { "id": 1, "username": "admin", "role": "admin" }
}
```

Токен передаётся заголовком `Authorization: Bearer <token>`.

## Запись (только админ)

| Метод и путь | Назначение |
| --- | --- |
| `POST /api/parking-lots` | Создать парковку |
| `PUT /api/parking-lots/:id` | Полностью заменить парковку |
| `DELETE /api/parking-lots/:id` | Удалить парковку (каскадом места и камеры) |
| `POST /api/parking-lots/:id/spots` | Создать место |
| `PUT /api/spots/:id` | Заменить место |
| `DELETE /api/spots/:id` | Удалить место |
| `POST /api/cameras` | Создать камеру |
| `PUT /api/cameras/:id` | Заменить камеру |
| `DELETE /api/cameras/:id` | Удалить камеру |

### Парковка

```json
{
  "name": "Парковка у корпуса",
  "address": "ул. Примерная, 1",
  "kind": "lot",
  "center_lat": 55.75,
  "center_lon": 37.6,
  "entrance_lat": 55.7501,
  "entrance_lon": 37.6002,
  "radius_m": 40,
  "geojson": { "type": "Polygon", "coordinates": [] }
}
```

Правила: `name` обязателен; `kind` — `lot` или `roadside`;
`center_lat`/`center_lon` обязательны; `entrance_*` — только парой;
`radius_m > 0`; `geojson` — валидный JSON.

### Место

```json
{
  "code": "T-04",
  "camera_id": 5,
  "polygon": [[0.168, 0.648], [0.194, 0.648], [0.194, 0.785], [0.168, 0.785]],
  "map_lat": 55.75009,
  "map_lon": 37.60009,
  "map_geojson": null
}
```

Правила: `code` обязателен и уникален в пределах парковки; `polygon` —
минимум 3 точки, нормализованные в `0..1`; `map_lat`/`map_lon` — только
парой и в допустимых диапазонах.

### Камера

```json
{
  "lot_id": 6,
  "name": "Тестовое видео",
  "source_type": "file",
  "source": "data/videos/sample_parking_fixed.mp4",
  "status": "offline",
  "fps": 12
}
```

Правила: `lot_id > 0`, `name` и `source` обязательны, `source_type` —
`rtsp`, `webcam` или `file`, `status` — `online`, `offline` или `error`.

## Внутренние эндпоинты CV-сервиса

Требуют заголовок `X-Internal-Token: <INTERNAL_TOKEN>`. Python не ходит
в БД напрямую, только через эти два вызова.

### `POST /api/internal/cameras/:id/heartbeat`

```json
{ "status": "online", "last_frame_at": "2026-11-17T11:36:07Z", "fps": 12, "resolution": "1280x720" }
```

`204 No Content`. `last_frame_at` не может быть в будущем (допуск 5 минут).
`last_frame_at` только увеличивается. При смене `status` всем подписчикам
лота уходит `camera_status`; при первом же heartbeat после простоя
эффективные статусы пересчитываются на стороне клиентов по инвалидации.

### `POST /api/internal/events`

```json
{
  "camera_id": 5,
  "events": [
    {
      "spot_id": 22,
      "status": "free",
      "confidence": 0.88,
      "captured_at": "2026-11-17T11:35:58Z",
      "snapshot_path": "data/snapshots/5/T-04_113558.jpg"
    }
  ]
}
```

```json
{ "accepted": 1 }
```

Правила: батч 1–500 событий, `status` — `free`, `occupied` или `unknown`,
`confidence` в `0..1`, `captured_at` не в будущем. События пишутся в
`spot_events`, текущее состояние — в `spot_status` (одна транзакция);
`last_frame_at` камеры обновляется по самому свежему `captured_at`.
После коммита по каждому событию уходит `spot_status`.

## WebSocket `/ws/live`

```
ws://localhost:8080/ws/live            # все лоты
ws://localhost:8080/ws/live?lot_id=6   # только лот 6
```

- Апгрейд разрешён только для Origin из `CORS_ORIGINS`; иначе `403`.
- Сервер шлёт `ping` каждые 54 с, ждёт `pong` до 60 с.
- Буфер клиента — 32 сообщения. Кто не успевает, отключается;
  клиент переподключается с экспоненциальной задержкой (1→10 с).
- Отдельного «снимка» через WS нет: начальное состояние берётся из REST,
  дальше приходят только изменения.

### `spot_status`

```json
{
  "type": "spot_status",
  "lot_id": 6,
  "spot_id": 22,
  "code": "T-04",
  "status": "free",
  "confidence": 0.88,
  "at": "2026-11-17T11:35:58Z"
}
```

### `camera_status`

```json
{
  "type": "camera_status",
  "lot_id": 6,
  "camera_id": 5,
  "status": "offline",
  "last_frame_at": "2026-11-17T11:35:58Z",
  "at": "2026-11-17T11:38:20Z"
}
```

`camera_status` приходит только при смене статуса камеры. Получив его,
фронтенд перезапрашивает парковку: смена статуса камеры меняет
эффективные статусы всех её мест.

## Примеры

```bash
# публичный список в границах карты, с местами
curl 'localhost:8080/api/parking-lots?bbox=37.59,55.749,37.61,55.751&include=spots'

# логин и создание места
TOKEN=$(curl -s localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"..."}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

curl -X POST localhost:8080/api/parking-lots/6/spots \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"code":"T-11","polygon":[[0.1,0.1],[0.2,0.1],[0.2,0.3],[0.1,0.3]]}'

# live-статусы лота 6
websocat 'ws://localhost:8080/ws/live?lot_id=6'
```
