# leotime

`leotime` is a lightweight, self-hosted time tracker inspired by the daily workflow of Solidtime, but intentionally smaller, faster to deploy, and easier to understand.

The project is being built as an open-source learning-friendly codebase for people coming from Python/Django who want to understand a Go API, a React frontend, SQLite, Docker-first deployment, and a test-heavy workflow.

## MVP Status (2026-07-08)

**The MVP is complete.** See [MVP delivery status](docs/33-mvp-delivery-status.md) for the full capability matrix and [Known gaps and audit](docs/34-known-gaps-and-audit.md) for open hardening items before production use at scale.

## Current Direction

- **Backend:** Go, Chi-style HTTP API, SQLite in WAL mode.
- **Frontend:** React, Vite, TypeScript.
- **Database:** SQLite file stored under `/data` in production.
- **Deployment:** Docker first, single container by default.
- **License:** MIT.
- **Languages:** Spanish and English.
- **Target user:** one owner/user first, with backlog support for later sync and desktop workflows.

## MVP Scope

The initial product target is:

- Normal login with a bootstrap admin user.
- Clients, projects, tasks, tags, time entries, rates, invoices, and reports.
- Calendar and weekly timesheet views.
- Multi-currency invoices that look official, including Spanish-style fiscal fields.
- Offline-capable web app that can create work locally and sync later.
- Configurable layout modes: Solidtime-like, minimal, compact.
- Docker-first VPS deployment.
- Unit, integration, UI, and end-to-end tests from the beginning.

## Repository Layout

```text
.
├── apps
│   ├── api        # Go backend, SQLite migrations, HTTP tests
│   └── web        # React/Vite frontend, Vitest and Playwright tests
├── deploy         # Production deployment examples
├── docs           # Product, architecture, backlog, testing, deployment, and Rust plan
├── Dockerfile     # Production image: builds web assets and Go binary
└── docker-compose.yml
```

## Local Development

API:

```bash
cd apps/api
go test ./...
go run ./cmd/leotime
```

Web:

```bash
cd apps/web
npm install
npm run dev
```

Docker:

```bash
cp .env.example .env.local   # optional; customize SMTP and bootstrap credentials
docker compose up --build
```

Without `.env.local`, Compose falls back to application defaults from `.env.example` values documented there.

## Continuous Integration

GitHub Actions runs on pushes and pull requests to `main`:

```bash
make fmt-check test-api-vet test-api test-web build-web test-e2e docker-build
```

See `.github/workflows/ci.yml` and `docs/05-testing-strategy.md`.

Install git hooks (recommended after clone):

```bash
make setup-hooks
```

The default bootstrap credentials are development-only:

- Email: `admin@example.com`
- Password: `change-me-now`

Change them with `LEOTIME_BOOTSTRAP_EMAIL` and `LEOTIME_BOOTSTRAP_PASSWORD`.

## Documentation Reading Order

Start with the [documentation index](docs/00-documentation-index.md), then:

1. [Product vision](docs/01-product-vision.md)
2. [MVP delivery status](docs/33-mvp-delivery-status.md)
3. [Known gaps and audit](docs/34-known-gaps-and-audit.md)
4. [Go architecture](docs/02-architecture-go.md)
5. [Data model](docs/03-data-model.md)
6. [Offline and sync strategy](docs/04-offline-sync.md)
7. [Testing strategy](docs/05-testing-strategy.md)
8. [VPS deployment](docs/06-deploy-vps.md)
9. [Rust + Axum alternative plan](docs/07-rust-axum-plan.md)
10. [Development workflow](docs/08-development-workflow.md)
11. [Solidtime import compatibility](docs/09-solidtime-import.md)
12. [Operations](docs/10-operations.md)
13. [Clients API](docs/11-clients-api.md)
14. [Implementation plan](docs/12-implementation-plan.md)
15. [Backlog](docs/13-backlog.md)
16. [Projects API](docs/14-projects-api.md)
17. [Solidtime-like UI theme](docs/15-solidtime-theme.md)
18. [Tasks API](docs/16-tasks-api.md)
19. [Tags API](docs/17-tags-api.md)
20. [Time Entries API](docs/18-time-entries-api.md)
21. [Timers API](docs/19-timers-api.md)
22. [Weekly timesheet](docs/20-weekly-timesheet.md)
23. [Calendar view](docs/21-calendar-view.md)
24. [Reports API](docs/22-reports-api.md)
25. [Invoices API](docs/23-invoices-api.md)
26. [Dashboard API](docs/24-dashboard-api.md)
27. [Theme selector](docs/25-theme-selector.md)
28. [Profile settings API](docs/26-profile-settings-api.md)
29. [Offline queue MVP](docs/27-offline-queue-mvp.md)
30. [Email notifications](docs/29-email-notifications.md)
31. [Password reset](docs/30-password-reset.md)
32. [S3 daily backups](docs/31-s3-daily-backups.md)
33. [API error responses](docs/32-api-errors.md)
34. [Billing documents (planned)](docs/32-billing-documents.md)
35. [ADR index](docs/adr/README.md)
36. [ADR 0001: Stack decision](docs/adr/0001-stack-go-sqlite-react.md)
37. [ADR 0002: In-process scheduler](docs/adr/0002-in-process-scheduler-outbox.md)
38. [ADR 0003: S3 backups](docs/adr/0003-s3-backup-encryption-and-restore.md)
39. [ADR 0004: Billing PDFs (planned)](docs/adr/0004-billing-documents-official-pdfs.md)
