SHELL := /bin/bash
.DEFAULT_GOAL := help

# Loads .env into the recipe's shell. Not `include .env`: make would keep the
# quotes in values such as SYNC_CRON.
WITH_ENV := set -a; [ -f .env ] && . ./.env; set +a;

BACKEND  := backend
FRONTEND := frontend

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ---------- dev ----------
.PHONY: setup
setup: ## Install dependencies and create .env
	@test -f .env || cp .env.example .env
	cd $(BACKEND) && go mod download
	cd $(FRONTEND) && npm ci

.PHONY: db
db: ## Start postgres and mailpit in docker
	docker compose up -d postgres mailpit

.PHONY: run-backend
run-backend: ## Run backend locally (needs `make db`)
	@$(WITH_ENV) export SMTP_HOST=$${SMTP_HOST:-localhost} SMTP_PORT=$${SMTP_PORT:-1025}; \
	 cd $(BACKEND) && go run ./cmd/tracker serve

.PHONY: run-frontend
run-frontend: ## Run Vite dev server on :5173 (proxies to :8080)
	cd $(FRONTEND) && npm run dev

.PHONY: storybook
storybook: ## Run Storybook on :6006
	cd $(FRONTEND) && npm run storybook

.PHONY: up
up: ## Build and start the whole stack (frontend on :3000)
	docker compose up -d --build

.PHONY: down
down: ## Stop the stack
	docker compose down

# ---------- database ----------
.PHONY: migrate-up
migrate-up: ## Apply migrations
	@$(WITH_ENV) cd $(BACKEND) && go run ./cmd/tracker migrate up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	@$(WITH_ENV) cd $(BACKEND) && go run ./cmd/tracker migrate down

.PHONY: migrate-new
migrate-new: ## Create a migration pair: make migrate-new name=add_artists
	@test -n "$(name)" || (echo "usage: make migrate-new name=<description>" && exit 1)
	@n=$$(ls $(BACKEND)/migrations/*.up.sql 2>/dev/null | wc -l | xargs); \
	 v=$$(printf "%06d" $$((n + 1))); \
	 touch $(BACKEND)/migrations/$${v}_$(name).up.sql $(BACKEND)/migrations/$${v}_$(name).down.sql; \
	 echo "created $(BACKEND)/migrations/$${v}_$(name).{up,down}.sql"

.PHONY: sqlc
sqlc: ## Generate Go code from SQL queries
	cd $(BACKEND) && go tool sqlc generate

# ---------- quality ----------
.PHONY: test
test: test-backend test-frontend ## Run all tests

.PHONY: test-backend
test-backend:
	cd $(BACKEND) && go test -race ./...

.PHONY: test-integration
test-integration: ## Run backend integration tests (PostgreSQL via testcontainers, needs docker)
	cd $(BACKEND) && go test -race -tags integration ./...

.PHONY: test-frontend
test-frontend:
	cd $(FRONTEND) && npm test

.PHONY: lint
lint: ## Run all linters
	cd $(BACKEND) && go vet ./... && (command -v golangci-lint >/dev/null && golangci-lint run || echo "golangci-lint not installed, skipped")
	cd $(FRONTEND) && npm run lint && npm run typecheck

.PHONY: fmt
fmt: ## Format code
	cd $(BACKEND) && gofmt -w .
	cd $(FRONTEND) && npm run format
