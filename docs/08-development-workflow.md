# Development Workflow

This document explains how to work on `leotime` if you are comfortable with Django and learning Go.

## Mental Model

```text
Django concept       leotime equivalent
settings.py          internal/config
urls.py              internal/httpapi/router.go
views.py             internal/httpapi handlers
models.py            migrations + store structs
services.py          internal/store methods
manage.py runserver  go run ./cmd/leotime
Django tests         go test ./...
```

Go does less implicit wiring. That is good for this project because the code should be easy to follow.

## Adding A Backend Feature

Example: adding clients.

1. Add or adjust a SQL migration.
2. Add store methods in `internal/store`.
3. Add HTTP handlers in `internal/httpapi`.
4. Add tests for the store and HTTP behavior.
5. Update frontend API calls.
6. Add UI tests for the new flow.
7. Update docs if the behavior changes product expectations.

## Adding A Frontend Feature

Example: adding a client list.

1. Add API functions in `src/lib/api.ts`.
2. Add a focused component under `src/components` or a feature folder.
3. Keep text keys in `src/lib/i18n.ts`.
4. Use stable layout dimensions so compact/minimal modes do not jump.
5. Add a Vitest test for rendering and interaction.
6. Add a Playwright test only when the browser workflow matters.

## Database Workflow

Migrations live in:

```text
apps/api/internal/db/migrations
```

They are embedded into the Go binary and applied on startup.

For now migrations are forward-only SQL files. If a migration is wrong during early development, create a new migration that fixes it instead of editing production history.

After changing backend routes or store behavior, restart the Go API process. `go run` does not hot-reload, so an old server can still answer `404 not found` for new endpoints such as `/restore`.

## Test Workflow

Run backend tests:

```bash
cd apps/api
go test ./...
```

Run frontend tests:

```bash
cd apps/web
npm test -- --run
```

Run E2E smoke:

```bash
cd apps/web
npm run test:e2e
```

Build Docker image:

```bash
docker compose build
```

## Local App

The easiest full-stack path is:

```bash
docker compose up --build
```

Then open:

```text
http://127.0.0.1:8080
```

Default local credentials:

```text
admin@example.com
change-me-now
```

Change them before exposing the app to the internet.

## Git Hooks

Install the repository pre-commit hook:

```bash
make setup-hooks
```

`make setup` also installs hooks automatically after `npm install`.

Before each commit, the hook runs:

```bash
make pre-commit
```

That gate verifies:

- Go formatting with `gofmt`.
- Go static analysis with `go vet`.
- Backend tests with `go test ./...`.
- Frontend unit tests with Vitest.
- Frontend production build with TypeScript and Vite.

Run the same gate manually at any time:

```bash
make pre-commit
```

AI agents must run `make pre-commit` before finishing any code change and fix failures before handoff. This is the same check the git hook runs on commit.

For full delivery checks after larger changes, also run:

```bash
make smoke
make deploy-check
```

