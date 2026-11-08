.PHONY: help up down logs backend cv frontend test

help:            ## Показать команды
	@grep -E '^[a-zA-Z_-]+:.*?## ' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

up:              ## Поднять всё в Docker
	docker compose -f deploy/docker-compose.yml up --build -d

down:            ## Остановить контейнеры
	docker compose -f deploy/docker-compose.yml down

logs:            ## Логи контейнеров
	docker compose -f deploy/docker-compose.yml logs -f

backend:         ## Запустить Go-бэкенд локально
	cd backend && go run ./cmd/api

cv:              ## Запустить CV-сервис локально
	cd cv && uvicorn app.main:app --reload --port 8000

frontend:        ## Запустить фронтенд локально
	cd frontend && npm run dev

test:            ## Тесты
	cd backend && go test ./...
	cd cv && pytest
