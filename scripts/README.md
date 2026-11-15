# scripts

Вспомогательные утилиты. Запускать из корня репозитория через venv CV-сервиса
(`cv/.venv/bin/python`).

## annotate_polygons.py — разметка мест по эталонному кадру

Обводит места мышью и сохраняет нормализованные полигоны в JSON, совместимый
с админским API и `data/lots/*.json`.

```bash
# кадр из видео на 12.5 секунде
cv/.venv/bin/python scripts/annotate_polygons.py \
  --video data/videos/pocket.mp4 --time 12.5 \
  --prefix A --start 1 \
  --output data/lots/pocket_spots.json

# готовый эталонный кадр
cv/.venv/bin/python scripts/annotate_polygons.py \
  --frame data/frames/reference.jpg \
  --output data/lots/pocket_spots.json

# быстрый режим: два противоположных угла вместо четырёх
cv/.venv/bin/python scripts/annotate_polygons.py \
  --frame data/frames/reference.jpg --mode rect
```

Управление:

- левый клик — добавить угол (место готово после `--points` кликов, по умолчанию 4);
- правый клик или `u` — убрать последний угол;
- `r` — сбросить текущее место, `d` — удалить последнее готовое;
- `g` — сетка 10 %, `s` — сохранить JSON, `q`/Esc — выход.

Выход:

```json
{
  "spots": [
    { "code": "A-01", "polygon": [[0.10, 0.55], [0.20, 0.55], [0.20, 0.78], [0.10, 0.78]] }
  ]
}
```

Координаты `map_lat/map_lon` скрипт не заполняет — их нужно добавить вручную
в JSON парковки или через админский API. Полигоны нормализованы в 0..1 и
проверяются теми же правилами, что и на бэкенде (≥3 точки, 0..1).

Тесты логики (без GUI): `cv/.venv/bin/python -m pytest cv/tests/test_annotation.py`.
