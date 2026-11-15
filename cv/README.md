# cv (Python)

Сервис компьютерного зрения: читает поток камеры, находит машины YOLO11,
сопоставляет их с полигонами мест и отправляет изменения занятости в Go-бэкенд.

## Поток данных

```
камера (RTSP / webcam / файл)
  → кадр (OpenCV), ресайз до MAX_FRAME_WIDTH
  → YOLO11: car / truck / bus / motorcycle
  → нормализация боксов в 0..1
  → сопоставление с полигонами мест (центр внутри или ≥60% углов)
  → гистерезис по кадрам (occupy/free)
  → POST /api/internal/events + снимки
  → POST /api/internal/cameras/{id}/heartbeat каждые 5 с
```

Сервис сам забирает полигоны мест из публичного API
`GET /api/parking-lots/{CV_LOT_ID}/spots`, поэтому разметку можно менять
без перезапуска кода (достаточно перезапустить воркер).

## Требования

- Python 3.11
- macOS / Linux / Windows
- 2–4 ГБ ОЗУ, CPU достаточно; CUDA-видеокарта ускоряет инференс
- Для macOS Apple MPS **не включён по умолчанию**: часть связок
  torch + ultralytics падает с нативным Objective-C исключением, которое
  нельзя перехватить из Python. Включайте только явно: `CV_DEVICE=mps`.

## Установка

Из корня репозитория:

```bash
make cv-setup     # python3.11 -m venv cv/.venv + pip install -e "cv[dev]"
make cv-model     # скачать yolo11n.pt в cv/models/
```

`make cv-setup` требует `python3.11` в `PATH`. На Windows создайте venv
вручную:

```powershell
py -3.11 -m venv cv\.venv
cv\.venv\Scripts\pip install -e "cv[dev]"
```

## Запуск

```bash
make cv           # FastAPI на http://localhost:8000 (uvicorn --reload)
make cv-run       # только воркер, без HTTP (удобно для отладки)
```

Для запуска нужны `CV_SOURCE`, `CV_CAMERA_ID`, `CV_LOT_ID` и `INTERNAL_TOKEN`
в `.env` (или в переменных окружения). Без них сервис поднимается в режиме
`idle` и ничего не делает.

Проверка:

```bash
curl localhost:8000/health   # {"status":"ok","worker":{...}}
curl localhost:8000/status   # подробное состояние воркера и мест
```

## Конфигурация

Все переменные читаются из окружения, `.env` в корне репозитория
загружается автоматически.

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `CV_SOURCE` | — | RTSP URL, номер веб-камеры или путь к файлу |
| `CV_SOURCE_TYPE` | `auto` | `auto`, `rtsp`, `webcam`, `file` |
| `CV_CAMERA_ID` | — | id камеры в БД (heartbeat и события) |
| `CV_LOT_ID` | — | id парковки, откуда брать полигоны мест |
| `INTERNAL_TOKEN` | — | общий секрет с Go-бэкендом |
| `CV_BACKEND_URL` | `http://localhost:8080` | адрес Go API |
| `CV_MODEL_PATH` | `models/yolo11n.pt` | веса YOLO (путь от `cv/`) |
| `CV_DEVICE` | `auto` | `auto` = CUDA→CPU; можно `cpu`, `0`, `mps` |
| `CONFIDENCE_THRESHOLD` | `0.4` | порог, при котором место считается занятым |
| `WEAK_CONFIDENCE_THRESHOLD` | `0.2` | нижний порог детекции: слабая детекция не занимает место, но не даёт освободить занятое |
| `SAMPLE_FPS` | `3` | сколько кадров в секунду обрабатывать |
| `MAX_FRAME_WIDTH` | `1280` | ресайз кадра перед детекцией |
| `OCCUPY_FRAMES` | `4` | сколько подряд кадров с машиной → `occupied` |
| `FREE_FRAMES` | `8` | сколько подряд пустых кадров → `free` |
| `CV_REALTIME` | `true` | проигрывать файл в реальном темпе |
| `CV_LOOP_VIDEO` | `false` | зацикливать файл (демо-режим) |
| `CV_HEARTBEAT_SECONDS` | `5` | период heartbeat |
| `CAMERA_RECONNECT_SECONDS` | `5` | пауза перед реконнектом RTSP |
| `CV_SAVE_OVERLAY` | `true` | писать размеченное видео |
| `CV_OVERLAY_PATH` | `data/videos/overlay.mp4` | путь оверлея (добавляется `_camera<id>`) |
| `SNAPSHOTS_DIR` | `data/snapshots` | снимки моментов смены статуса |
| `CV_LOG_LEVEL` | `INFO` | уровень логирования |

Пути `CV_SOURCE`, `CV_OVERLAY_PATH`, `SNAPSHOTS_DIR` считаются от корня
репозитория; `CV_MODEL_PATH` — от каталога `cv/`.

## Алгоритм занятости

1. Детектор возвращает боксы машин в пикселях; они нормируются в 0..1,
   чтобы не зависеть от разрешения потока.
2. Место считается занятым, если центр бокса внутри полигона **или**
   хотя бы 3 из 5 контрольных точек бокса (центр + 4 угла) попали внутрь.
3. Порог детекции асимметричный: `occupied` ставится только по детекции
   не ниже `CONFIDENCE_THRESHOLD` (0.4), а детекция от `WEAK_CONFIDENCE_THRESHOLD`
   (0.2) уже мешает перевести место в `free`. Так проезжающая машина,
   которая частично перекрыла припаркованную, не «освобождает» место.
4. Гистерезис: `occupied` ставится после `OCCUPY_FRAMES` подряд кадров
   с машиной, `free` — после `FREE_FRAMES` подряд кадров без детекций
   (даже слабых). Короткие пропуски детекции не меняют статус.
5. Событие отправляется только при смене статуса, поэтому история
   в `spot_events` не засоряется одинаковыми записями.
6. Для каждого события сохраняется снимок кадра с разметкой в
   `data/snapshots/camera<id>_spot<id>_<time>.jpg`.

При старте воркер ставит камеру `online`, при потере потока — `offline`
и пытается переподключиться. При остановке — `offline`.

## Внутренний API Go

Защищён заголовком `X-Internal-Token` (значение `INTERNAL_TOKEN`):

- `POST /api/internal/cameras/{id}/heartbeat`
  ```json
  {"status":"online","last_frame_at":"2026-11-14T10:00:00Z","fps":3,"resolution":"1920x1080"}
  ```
- `POST /api/internal/events`
  ```json
  {"camera_id":5,"events":[
    {"spot_id":21,"status":"occupied","confidence":0.87,
     "captured_at":"2026-11-14T10:00:00Z",
     "snapshot_path":"data/snapshots/camera5_spot21_....jpg"}
  ]}
  ```

Go проверяет токен, валидирует тело и в одной транзакции пишет
`spot_events`, обновляет `spot_status` и `cameras.last_frame_at`.

## Демо-режим на видеофайле

```bash
CV_SOURCE=data/videos/sample_parking.webm \
CV_SOURCE_TYPE=file \
CV_CAMERA_ID=5 \
CV_LOT_ID=6 \
CV_REALTIME=true \
CV_LOOP_VIDEO=true \
make cv-run
```

Для демонстрации на защите лучше использовать реальную RTSP-камеру.
Файл — только резервный сценарий, когда камера недоступна.

## Разметка полигонов

Полигоны мест задаются в нормализованных координатах 0..1 и не зависят от
разрешения потока. Быстрый способ получить их — обвести места мышью на
эталонном кадре:

```bash
cv/.venv/bin/python scripts/annotate_polygons.py \
  --frame data/frames/reference.jpg \
  --prefix A --start 1 \
  --output data/lots/pocket_spots.json
```

Скрипт пишет JSON `{"spots": [{"code", "polygon"}]}`; координаты
`map_lat/map_lon` добавляются отдельно. Подробности — в `scripts/README.md`.

## Тесты

```bash
cd cv && .venv/bin/python -m pytest -q
```

Покрыты: парсинг и проверка полигонов, point-in-polygon, сопоставление
боксов, гистерезис (включая дребезг и слабые детекции), определение типа
источника, подготовка полигонов для разметки.
