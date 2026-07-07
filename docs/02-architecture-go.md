# Go Architecture

The Go backend is intentionally small and explicit. The goal is not to hide the framework. The goal is to make the data flow easy to read.

## Main Runtime Flow

```text
main.go
  -> load config from environment
  -> open SQLite database
  -> apply embedded migrations
  -> create bootstrap admin if needed
  -> start background scheduler (timer mail scan + outbox processing)
  -> build HTTP router
  -> serve API and static frontend
```

## Package Layout

```text
apps/api
├── cmd/leotime        # executable entrypoint
└── internal
    ├── auth           # password hashing and verification
    ├── config         # environment parsing
    ├── db             # SQLite open and migration runner
    ├── httpapi        # routes, handlers, JSON responses
    ├── mail           # SMTP and log senders
    ├── metrics        # Prometheus counters for scheduler/mail
    ├── notify         # still-running timer notification jobs
    ├── outbox         # durable email queue + retry processor
    ├── scheduler      # in-process scan/outbox tickers
    └── store          # database-backed business operations
```

## Why This Shape

This is close to a Django mental model, but with less magic:

- `cmd/leotime/main.go` is similar to a Django `manage.py` entrypoint plus ASGI/WSGI boot.
- `internal/config` is similar to `settings.py`, but environment-first.
- `internal/db/migrations` is similar to Django migrations, but SQL files are explicit.
- `internal/store` is similar to simple service/query classes.
- `internal/httpapi` is similar to views/controllers.

## Request Lifecycle

```text
Browser
  -> HTTP router
  -> handler
  -> store method
  -> SQLite query
  -> JSON response
```

Authentication uses an HTTP-only cookie backed by a `sessions` table. Passwords are not stored directly. The current scaffold uses PBKDF2-HMAC-SHA256 with random salts and constant-time comparison so the code remains dependency-light and understandable.

## SQLite Policy

SQLite runs in WAL mode. For the first single-user version, the database pool is conservative and uses one open connection. That keeps correctness easy to reason about. If the product grows, we can later relax this and tune the pool.

Every connection should keep these expectations:

- Foreign keys enabled.
- Busy timeout configured.
- WAL journal mode.
- Normal synchronous mode for good write performance with acceptable durability for a backed-up VPS app.

## API Versioning

The API starts under `/api/v1`.

The initial routes are:

- `GET /api/health`
- `GET /api/v1/session`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/overview`
- `GET /api/v1/clients`
- `POST /api/v1/clients`
- `GET /api/v1/clients/{clientID}`
- `PATCH /api/v1/clients/{clientID}`
- `DELETE /api/v1/clients/{clientID}`
- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/projects/{projectID}`
- `PATCH /api/v1/projects/{projectID}`
- `DELETE /api/v1/projects/{projectID}`

More feature routes should be added by domain:

- `/api/v1/tasks`
- `/api/v1/tags`
- `/api/v1/time-entries`
- `/api/v1/reports`
- `/api/v1/invoices`
- `/api/v1/sync`
