# GuildChat — common tasks
# Run `make` or `make help` to see everything.

DC      := docker compose
APP_DIR := app

.DEFAULT_GOAL := help
.PHONY: help up down restart build rebuild logs logs-db ps sh psql reset seed test vet fmt tidy run health

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

## ---------- docker compose ----------

up: ## Start the stack in the background
	$(DC) up -d

down: ## Stop the stack (keeps the database volume)
	$(DC) down

restart: ## Restart just the backend
	$(DC) restart backend

build: ## Build the backend image
	$(DC) build

rebuild: ## Rebuild from scratch and start
	$(DC) build --no-cache
	$(DC) up -d

logs: ## Tail backend logs
	$(DC) logs -f backend

logs-db: ## Tail postgres logs
	$(DC) logs -f postgres

ps: ## Show container status
	$(DC) ps

sh: ## Shell into the backend container
	$(DC) exec backend sh

psql: ## Open a psql prompt on the database
	$(DC) exec postgres psql -U postgres -d guildchat

## ---------- database ----------

reset: ## DESTROY the database volume and re-run db/init.sql
	$(DC) down -v
	$(DC) up -d

## ---------- go (runs on the host) ----------

test: ## Run go tests
	cd $(APP_DIR) && go test ./...

vet: ## Run go vet
	cd $(APP_DIR) && go vet ./...

fmt: ## Format go code
	cd $(APP_DIR) && go fmt ./...

tidy: ## Tidy go.mod / go.sum
	cd $(APP_DIR) && go mod tidy

run: ## Run the server directly on the host (needs postgres up)
	cd $(APP_DIR) && go run ./cmd/server

## ---------- misc ----------

health: ## Curl the health endpoint
	@curl -fsS localhost:8080/health && echo
