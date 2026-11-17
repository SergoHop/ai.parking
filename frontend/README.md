# frontend

Веб-интерфейс: карта парковок и live-статусы мест.

Стек: React 19 + TypeScript + Vite 7, данные — TanStack Query,
состояние выбора — Zustand. Карта — Яндекс Карты JS API 3.0
(ключ в `.env`), до получения ключа работает схематичный слой.

```
src/
├── api/            — REST-клиент и WebSocket с реконнектом
├── features/
│   └── parking-map — карта (mock и Яндекс), панель лота, live-обновления
├── pages/          — MapPage: карта + панель
├── store/          — Zustand: выбранный лот/место, показ занятых
├── styles/         — mobile-first CSS
├── types/          — типы ответов API и объявления ymaps3
├── App.tsx
└── main.tsx
```

## Команды

```bash
make frontend-install   # npm install
make frontend           # dev-сервер http://localhost:5173
make frontend-build     # tsc --noEmit + vite build
```

Можно и напрямую: `npm install`, `npm run dev`, `npm run build`.
Если `npm` не в `PATH`, Makefile подставит `~/.local/node/bin`.

## Переменные окружения

Читаются из корневого `.env` (`envDir: '..'` в `vite.config.ts`):

| Переменная | Назначение |
| --- | --- |
| `VITE_API_URL` | База REST, по умолчанию `http://localhost:8080` |
| `VITE_WS_URL` | Адрес live-статусов, по умолчанию `ws://localhost:8080/ws/live` |
| `VITE_YANDEX_MAPS_API_KEY` | Ключ JS API 3.0 с referer `localhost`/`127.0.0.1` |

Без ключа включается `MockMap`: те же координаты и живые данные,
проекция с учётом широты, без запросов к Яндексу.

## Поведение карты

- Список парковок и мест обновляется раз в 30 с; изменения приходят
  по WebSocket и применяются к кэшу без перезагрузки.
- Статусы старше `FRESHNESS_TTL_SECONDS` и места offline-камеры
  показываются серыми («нет данных»), а не свободными.
- Занятые места скрыты по умолчанию; `?debug=1` показывает их,
  добавляет легенду и переключатель.
