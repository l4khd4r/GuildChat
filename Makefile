# GuildChat — common tasks
# Run `make` or `make help` to see everything.

DC      := docker compose
APP_DIR := app

.DEFAULT_GOAL := help
.PHONY: help up down restart build rebuild logs logs-db ps sh psql reset seed test vet fmt tidy run health \
        migrate-up migrate-down migrate-version migrate-force migrate-new keys

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

reset: ## DESTROY the database volume and re-migrate from scratch
	$(DC) down -v
	$(DC) up -d

migrate-up: ## Apply pending migrations (the server also does this on boot)
	cd $(APP_DIR) && go run ./cmd/migrate up

migrate-down: ## Roll back migrations: make migrate-down N=1 (omit N for all)
	cd $(APP_DIR) && go run ./cmd/migrate down $(N)

migrate-version: ## Show the current schema version
	cd $(APP_DIR) && go run ./cmd/migrate version

migrate-force: ## Clear a dirty schema at a version: make migrate-force V=1
	@test -n "$(V)" || { echo "usage: make migrate-force V=<version>"; exit 1; }
	cd $(APP_DIR) && go run ./cmd/migrate force $(V)

migrate-new: ## Create an empty migration pair: make migrate-new NAME=add_guilds
	@test -n "$(NAME)" || { echo "usage: make migrate-new NAME=<name>"; exit 1; }
	@dir=$(APP_DIR)/internal/database/migrations; \
	next=$$(ls $$dir | sed -n 's/^\([0-9]\{6\}\)_.*/\1/p' | sort -n | tail -1); \
	next=$$(printf '%06d' $$((10#$${next:-0} + 1))); \
	touch $$dir/$${next}_$(NAME).up.sql $$dir/$${next}_$(NAME).down.sql; \
	echo "created $$dir/$${next}_$(NAME).{up,down}.sql"

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

## ---------- keys ----------

keys: ## Generate the RSA keypair in keys/ (make keys FORCE=1 to overwrite)
	@if [ -e keys/private.pem ] || [ -e keys/public.pem ]; then \
		test -n "$(FORCE)" || { echo "keys/ already has a keypair; re-run with FORCE=1 to overwrite"; exit 1; }; \
	fi
	@mkdir -p keys
	@openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out keys/private.pem
	@openssl rsa -in keys/private.pem -pubout -out keys/public.pem
	@chmod 600 keys/private.pem
	@chmod 644 keys/public.pem
	@echo "wrote keys/private.pem and keys/public.pem"

## ---------- misc ----------

health: ## Curl the health endpoint
	@curl -fsS localhost:8080/health && echo
