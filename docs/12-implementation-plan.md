# Implementation Plan

This is the living implementation plan for `leotime`. It turns the product vision into small, reviewable delivery slices.

## Current Rule

Every product slice should include:

- Backend store/service code.
- Authenticated HTTP API.
- Frontend API client.
- UI with validation and empty/loading/error states.
- Unit or integration tests for risky behavior.
- Documentation update.
- A focused commit.

## Completed Foundation

- Monorepo scaffold with Go API, React/Vite web app, SQLite, Docker, and docs.
- Repo-scoped Codex guidance, skills, and agents.
- Docker-first operational tooling through `Makefile`.
- Prometheus metrics endpoint and observability profile.
- Solidtime ZIP import foundation with parser, importer, compatibility tables, and tests.
- Client CRUD with backend API, frontend workbench, validation, tests, and docs.
- Project CRUD with optional client assignment, colors, optional rate override, tests, and docs.
- Task CRUD with optional project assignment, billable default, archive behavior, tests, and docs.
- Tag CRUD with unique names, colors, hard delete, tests, and docs.
- Manual time entries with one-minute precision, billable flag, overlap warnings, duration calculation, tests, and docs.
- Timer workflow with start/stop, multiple open timers, overlap warnings, live clock UI, tests, and docs.
- Weekly timesheet with 7-day grid, week navigation, API date filters, inline editing, tests, and docs.
- Calendar view with monthly grid, day selection, month navigation, inline editing, tests, and docs.

## MVP Delivery Slices

1. **Reports And Exports**
   - CSV and JSON exports.
   - Hide start/end timestamps when the report is configured to show only totals.

2. **Invoices**
   - Draft invoices from billable time.
   - Multi-currency, frozen client fields, line items, tax/withholding fields, and PDF/export.

3. **Offline Queue MVP**
   - Offline creation/edit queue for the core daily workflow.
   - Syncs when connectivity returns.

## Quality Gates By Slice

Small backend-only slice:

```bash
make test-api
```

Frontend slice:

```bash
make test-web
npm --workspace @leotime/web run build
```

Full product slice:

```bash
make test
make test-e2e
make docker-build
make smoke
```

When port `8080` is already occupied by a local process without static assets, run smoke against a temporary app instance with `BASE_URL`.

## Current Next Task

The next implementation slice is **Reports And Exports**.
