.DEFAULT_GOAL := help

COMPOSE ?= docker compose
GO ?= go
NPM ?= npm
SERVICE ?=

.PHONY: help up down restart build logs ps config test vet web-install typecheck check db-shell

help: ## Показать доступные команды
	@awk 'BEGIN {FS = ":.*## "; printf "Использование: make <команда>\n\n"} /^[a-zA-Z_-]+:.*## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

up: ## Собрать и запустить приложение в фоне, дождаться готовности
	$(COMPOSE) up --build --detach --wait
	@printf 'Приложение: http://localhost:%s\n' "$${APP_PORT:-8080}"

down: ## Остановить и удалить контейнеры, сохранив данные MySQL
	$(COMPOSE) down

restart: ## Перезапустить контейнеры (можно указать SERVICE=api)
	$(COMPOSE) restart $(SERVICE)

build: ## Собрать Docker-образы API и веб-приложения
	$(COMPOSE) build

logs: ## Смотреть логи (можно указать SERVICE=api)
	$(COMPOSE) logs --follow --tail=100 $(SERVICE)

ps: ## Показать состояние контейнеров
	$(COMPOSE) ps

config: ## Проверить конфигурацию Docker Compose
	$(COMPOSE) config --quiet

test: ## Запустить Go-тесты локально (нужен Go из go.mod)
	$(GO) test ./...

vet: ## Проверить Go-код статическим анализатором
	$(GO) vet ./...

web-install: ## Установить зависимости фронтенда локально
	$(NPM) --prefix web ci --no-audit --no-fund

typecheck: ## Проверить TypeScript (сначала make web-install)
	$(NPM) --prefix web run typecheck

check: config test vet typecheck ## Проверить Compose, Go и TypeScript

db-shell: ## Открыть MySQL-консоль в работающем контейнере
	$(COMPOSE) exec db sh -c 'MYSQL_PWD="$$MYSQL_PASSWORD" exec mysql --protocol=TCP -h127.0.0.1 -u"$$MYSQL_USER" "$$MYSQL_DATABASE"'
