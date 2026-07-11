# Go Architecture

The Go backend is intentionally small and explicit. The goal is not to hide the framework. The goal is to make the data flow easy to read.

## Main Runtime Flow

```text
main.go
  -> load config from environment
  -> open SQLite database
  -> apply embedded migrations
  -> create bootstrap admin if needed
  -> wire backup service, email outbox, notifier, scheduler
  -> build HTTP router
  -> serve API and static frontend
```

## Package Layout

```text
apps/api
├── cmd/leotime        # server, seed, import solidtime, backup subcommands
└── internal
    ├── apierr         # structured JSON error types
    ├── auth           # password hashing and verification
    ├── backup         # snapshot, S3 storage, restore, scheduler hooks
    ├── config         # environment parsing
    ├── db             # SQLite open and migration runner
    ├── httpapi        # routes, handlers, JSON responses
    ├── mail           # SMTP and log senders
    ├── metrics        # Prometheus counters
    ├── notify         # timer, backup, password-reset mail builders
    ├── outbox         # durable email queue + retry processor
    ├── scheduler      # in-process scan/outbox/backup tickers
    ├── seed           # demo data loader
    ├── solidtimeimport
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
  -> JSON response (or structured error envelope)
```

Authentication uses an HTTP-only cookie backed by a `sessions` table. Passwords use PBKDF2-HMAC-SHA256 with random salts.

API errors return `{ "error": { "code", "message", "fields?" } }`. See [API error responses](32-api-errors.md).

## SQLite Policy

SQLite runs in WAL mode with a conservative single-connection pool for the first single-user version. Foreign keys, busy timeout, and WAL are enabled on every connection.

## API Surface

Health and metrics:

- `GET /api/health`
- `GET /metrics` (Prometheus; protect in production—see doc 34)

Auth:

- `GET /api/v1/session`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

Core resources (all under `/api/v1`, cookie auth):

- `overview`, `clients`, `projects`, `tasks`, `tags`
- `time-entries`, `timers`
- `reports/time`, `invoices`
- `dashboard/stats`
- `profile` (GET/PATCH), `profile/password`
- `import/solidtime`
- `backups/settings`, `backups/test`, `backups/run`, `backups/restore`, `backups/status`, `backups/objects`

Full per-resource docs: [documentation index](00-documentation-index.md).

## Frontend Layout

```text
apps/web/src
├── App.tsx              # session boot, auth gate
├── features/
│   ├── shell/           # DashboardShell, routing, sidebar
│   ├── clients/         # ClientPanel
│   ├── projects/        # ProjectPanel
│   ├── tasks/           # TaskPanel
│   └── tags/            # TagPanel
└── lib/                 # api, i18n, offline, *Ui panels
```

## Architecture decisions

| ADR | Topic | Implemented |
| --- | --- | --- |
| [0001](adr/0001-stack-go-sqlite-react.md) | Stack | Yes |
| [0002](adr/0002-in-process-scheduler-outbox.md) | Scheduler + outbox | Yes |
| [0003](adr/0003-s3-backup-encryption-and-restore.md) | S3 backup/restore | Yes |
| [0004](adr/0004-billing-documents-official-pdfs.md) | Official invoice PDFs | **Partial; hardening queued** |

Index: [adr/README.md](adr/README.md).

## CLI Subcommands

| Command | Purpose |
| --- | --- |
| `(default)` | HTTP server |
| `seed` | Demo data |
| `import solidtime` | ZIP import |
| `backup run \| list \| restore` | Backup ops |

See [Operations](10-operations.md) and [MVP delivery status](33-mvp-delivery-status.md).
