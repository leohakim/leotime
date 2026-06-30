# leotime

`leotime` is a lightweight, self-hosted time tracker inspired by the daily workflow of Solidtime, but intentionally smaller, faster to deploy, and easier to understand.

The project is being built as an open-source learning-friendly codebase for people coming from Python/Django who want to understand a Go API, a React frontend, SQLite, Docker-first deployment, and a test-heavy workflow.

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
docker compose up --build
```

The default bootstrap credentials are development-only:

- Email: `admin@example.com`
- Password: `change-me-now`

Change them with `LEOTIME_BOOTSTRAP_EMAIL` and `LEOTIME_BOOTSTRAP_PASSWORD`.

## Documentation Reading Order

1. [Product vision](docs/01-product-vision.md)
2. [Go architecture](docs/02-architecture-go.md)
3. [Data model](docs/03-data-model.md)
4. [Offline and sync strategy](docs/04-offline-sync.md)
5. [Testing strategy](docs/05-testing-strategy.md)
6. [VPS deployment](docs/06-deploy-vps.md)
7. [Rust + Axum alternative plan](docs/07-rust-axum-plan.md)
8. [Development workflow](docs/08-development-workflow.md)
9. [Solidtime import compatibility](docs/09-solidtime-import.md)
10. [Operations](docs/10-operations.md)
11. [Clients API](docs/11-clients-api.md)
12. [Implementation plan](docs/12-implementation-plan.md)
13. [Backlog](docs/13-backlog.md)
14. [Projects API](docs/14-projects-api.md)
15. [Solidtime-like UI theme](docs/15-solidtime-theme.md)
16. [ADR 0001: Stack decision](docs/adr/0001-stack-go-sqlite-react.md)
