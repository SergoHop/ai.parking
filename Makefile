-include .env

COMPOSE := docker compose --env-file .env -f deploy/docker-compose.yml

POSTGRES_USER ?= parking
POSTGRES_PASSWORD ?= parking
POSTGRES_DB ?= parking
POSTGRES_PORT ?= 5432
MIGRATE_DB_URL = postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable

.PHONY: help up down logs ps rebuild backend cv cv-setup cv-model cv-run frontend frontend-install frontend-build test vet tidy psql migrate-up migrate-down seed seed-reset

CV_PY := cv/.venv/bin/python

# Node может быть не в PATH (например, портативная сборка в ~/.local/node).
NODE_BIN := $(shell command -v node 2>/dev/null || echo "$(HOME)/.local/node/bin/node")
NPM := $(if $(shell command -v npm 2>/dev/null),npm,$(HOME)/.local/node/bin/npm)

help:            ## Показать команды
	@grep -E '^[a-zA-Z_-]+:.*?## ' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

up:              ## Поднять Postgres, применить миграции и запустить backend
	$(COMPOSE) up --build -d

down:            ## Остановить контейнеры
	$(COMPOSE) down

logs:            ## Логи контейнеров
	$(COMPOSE) logs -f

ps:              ## Статус контейнеров
	$(COMPOSE) ps

rebuild:         ## Пересобрать backend и перезапустить
	$(COMPOSE) up --build -d backend

backend:         ## Запустить Go-бэкенд локально
	cd backend && go run ./cmd/api

cv-setup:        ## Создать venv и установить зависимости CV (Python 3.11)
	python3.11 -m venv cv/.venv
	$(CV_PY) -m pip install -U pip
	$(CV_PY) -m pip install -e "cv[dev]"

cv-model:        ## Скачать веса yolo11n.pt в cv/models
	mkdir -p cv/models
	$(CV_PY) -c "from ultralytics import YOLO; YOLO('cv/models/yolo11n.pt')"

cv:              ## Запустить CV-сервис локально
	cd cv && .venv/bin/python -m uvicorn app.main:app --reload --port 8000

cv-run:          ## Запустить CV-воркер без FastAPI (отладка)
	cd cv && .venv/bin/python -m app.pipeline.runner

frontend:        ## Запустить фронтенд локально
	cd frontend && PATH="$(dir $(NODE_BIN)):$$PATH" $(NPM) run dev

frontend-install: ## Установить зависимости фронтенда
	cd frontend && PATH="$(dir $(NODE_BIN)):$$PATH" $(NPM) install

frontend-build:  ## Проверить типы и собрать фронтенд
	cd frontend && PATH="$(dir $(NODE_BIN)):$$PATH" $(NPM) run build

test:            ## Тесты
	cd backend && go test ./...
	cd cv && if [ -x .venv/bin/python ]; then .venv/bin/python -m pytest; else python3 -m pytest; fi

vet:             ## go vet
	cd backend && go vet ./...

tidy:            ## go mod tidy + gofmt
	cd backend && go mod tidy && gofmt -w .

psql:            ## Открыть psql в контейнере postgres
	$(COMPOSE) exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

seed:            ## Заполнить БД демо-парковкой из data/lots/demo.json
	cd backend && go run ./cmd/seed -file ../data/lots/demo.json

seed-reset:      ## Пересоздать демо-парковку (удаляет старую с тем же именем)
	cd backend && go run ./cmd/seed -file ../data/lots/demo.json -force

migrate-up:      ## Применить миграции
	$(COMPOSE) run --rm migrate

migrate-down:    ## Откатить последнюю миграцию
	$(COMPOSE) run --rm --entrypoint migrate migrate \
		-path=/migrations -database "$(MIGRATE_DB_URL)" down 1
