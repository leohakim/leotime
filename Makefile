.PHONY: test test-api test-web build-web dev-api dev-web docker-up docker-down

test: test-api test-web

test-api:
	cd apps/api && go test ./...

test-web:
	cd apps/web && npm test -- --run

build-web:
	cd apps/web && npm run build

dev-api:
	cd apps/api && go run ./cmd/leotime

dev-web:
	cd apps/web && npm run dev

docker-up:
	docker compose up --build

docker-down:
	docker compose down

