.DEFAULT_GOAL := help

ZIP ?=
USER_EMAIL ?= admin@example.com
BASE_URL ?= http://127.0.0.1:8080
K6_BASE_URL ?= http://leotime:8080
K6_VUS ?= 10
K6_DURATION ?= 30s

.PHONY: help setup dev dev-api dev-web up down logs migrate test test-api test-web test-e2e build-web smoke bench stress metrics docker-build deploy-check import-solidtime import-solidtime-dry

help: ## 🧭 Show available commands
	@printf "\n🕒 leotime developer commands\n\n"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\n"

setup: ## 🧰 Install local dependencies
	@printf "🧰 Installing web dependencies...\n"
	npm install
	@printf "✅ Setup complete\n"

dev: ## 🚀 Run API and web dev servers in parallel
	@printf "🚀 Starting API and web dev servers...\n"
	@trap 'kill 0' INT TERM EXIT; \
	(cd apps/api && go run ./cmd/leotime) & \
	(cd apps/web && npm run dev) & \
	wait

dev-api: ## 🧪 Run only the Go API
	@printf "🧪 Starting Go API...\n"
	cd apps/api && go run ./cmd/leotime

dev-web: ## 🎨 Run only the web app
	@printf "🎨 Starting Vite...\n"
	cd apps/web && npm run dev

up: ## 🐳 Start the Docker stack
	@printf "🐳 Starting Docker stack...\n"
	docker compose up --build -d

down: ## 🛑 Stop the Docker stack
	@printf "🛑 Stopping Docker stack...\n"
	docker compose down

logs: ## 📜 Tail application logs
	@printf "📜 Tailing logs...\n"
	docker compose logs -f leotime

migrate: ## 🗄️ Apply migrations by starting the API once
	@printf "🗄️ Applying migrations through application startup...\n"
	cd apps/api && go run ./cmd/leotime -migrate-only

test: test-api test-web ## ✅ Run backend and frontend tests

test-api: ## ✅ Run Go tests
	@printf "✅ Running Go tests...\n"
	cd apps/api && go test ./...

test-web: ## ✅ Run frontend unit tests
	@printf "✅ Running frontend tests...\n"
	npm --workspace @leotime/web test -- --run

test-e2e: ## 🧭 Run Playwright E2E tests
	@printf "🧭 Running Playwright tests...\n"
	npm --workspace @leotime/web run test:e2e

build-web: ## 📦 Build frontend assets
	@printf "📦 Building web app...\n"
	npm --workspace @leotime/web run build

smoke: ## 💨 Smoke test API, session, frontend, and metrics
	@printf "💨 Smoke testing $(BASE_URL)...\n"
	@curl -fsS "$(BASE_URL)/api/health" >/dev/null
	@curl -fsS "$(BASE_URL)/api/v1/session" >/dev/null
	@curl -fsS "$(BASE_URL)/metrics" >/dev/null
	@curl -fsS "$(BASE_URL)/" >/dev/null
	@printf "✅ Smoke checks passed\n"

bench: ## ⏱️ Run Go benchmarks
	@printf "⏱️ Running Go benchmarks...\n"
	cd apps/api && go test -bench=. ./...

stress: ## 🔥 Run k6 stress tests through Docker
	@printf "🔥 Running k6 stress test against $(K6_BASE_URL)...\n"
	docker compose up -d leotime
	docker compose --profile tools run --rm -e BASE_URL="$(K6_BASE_URL)" -e K6_VUS="$(K6_VUS)" -e K6_DURATION="$(K6_DURATION)" k6 run /scripts/leotime-smoke.js

metrics: ## 📈 Start Prometheus and Grafana profile
	@printf "📈 Starting observability stack...\n"
	docker compose --profile observability up -d prometheus grafana
	@printf "✅ Prometheus: http://127.0.0.1:9090\n"
	@printf "✅ Grafana:    http://127.0.0.1:3001\n"

docker-build: ## 🏗️ Build production Docker image
	@printf "🏗️ Building Docker image...\n"
	docker compose build

deploy-check: test build-web docker-build ## 🚢 Run deploy readiness checks
	@printf "🚢 Deploy checks completed\n"

import-solidtime: ## 📥 Import Solidtime ZIP into local API database
	@test -n "$(ZIP)" || (printf "❌ Usage: make import-solidtime ZIP=/path/export.zip USER_EMAIL=$(USER_EMAIL)\n" && exit 2)
	@printf "📥 Importing Solidtime export...\n"
	cd apps/api && go run ./cmd/leotime import solidtime --file "$(ZIP)" --user-email "$(USER_EMAIL)"

import-solidtime-dry: ## 🔎 Dry-run Solidtime ZIP import
	@test -n "$(ZIP)" || (printf "❌ Usage: make import-solidtime-dry ZIP=/path/export.zip USER_EMAIL=$(USER_EMAIL)\n" && exit 2)
	@printf "🔎 Dry-running Solidtime import...\n"
	cd apps/api && go run ./cmd/leotime import solidtime --file "$(ZIP)" --user-email "$(USER_EMAIL)" --dry-run
