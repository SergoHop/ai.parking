-include .env

COMPOSE := docker compose --env-file .env -f deploy/docker-compose.yml

POSTGRES_USER ?= parking
POSTGRES_PASSWORD ?= parking
POSTGRES_DB ?= parking
POSTGRES_PORT ?= 5432
MIGRATE_DB_URL = postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable

.PHONY: help up down logs ps rebuild backend cv frontend test vet tidy psql migrate-up migrate-down seed seed-reset

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

cv:              ## Запустить CV-сервис локально
	cd cv && uvicorn app.main:app --reload --port 8000

frontend:        ## Запустить фронтенд локально
	cd frontend && npm run dev

test:            ## Тесты
	cd backend && go test ./...
	cd cv && pytest

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
