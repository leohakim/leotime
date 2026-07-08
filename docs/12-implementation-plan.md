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
- Reports and exports with grouped totals, CSV/JSON export, optional timestamps, tests, and docs.
- Invoices with draft-from-time, frozen client fields, tax/withholding, status workflow, HTML/CSV/JSON export, tests, and docs.
- Dashboard UI with recent entries, last 7 days, activity heatmap, weekly bars, billable totals, project donut, tests, and docs.
- Theme selector with Solidtime, light, dark, and minimal palettes, persistence, tests, and docs.
- Profile settings with account update, password change, timezone, currency, theme sync, tests, and docs.
- Offline queue MVP with IndexedDB mutation queue, optimistic UI, auto-sync, and daily workflow coverage.
- Email outbox foundation with SMTP/log senders, SQLite outbox, and retry policy (migration `000005`).
- In-process scheduler for still-running timer notifications with Prometheus metrics.
- Timer notification settings in profile API and Settings UI.
- Password reset email with one-time tokens, outbox delivery, and login/reset UI.
- S3 daily backups with encrypted credentials, scheduler, CLI, restore, and Settings UI.
- Backup/restore email notifications with profile toggles and outbox delivery.
- Dev seed command for demo clients, projects, tasks, tags, and time entries.
- GitHub Actions CI for API tests, web build, Playwright, Docker build, and smoke checks.

## Post-MVP Delivery Slices

| Slice | Status | Notes |
| --- | --- | --- |
| Timer notification settings UI | Done | Profile API + Ajustes toggle and hours field |
| Solidtime import `still_active_email_sent_at` | Done | Persisted on import create/update |
| Password reset email | Done | Outbox mail + login/reset UI |
| S3 daily backups + restore | Done | Spec in `docs/31-s3-daily-backups.md`; 01:00 default, 365d retention, UI + CLI + scheduler |
| Backup/restore email notifications | Done | Profile toggles, outbox kinds, localized templates; see `docs/29-email-notifications.md` |
| Seed/dev data command | Done | `make seed` / `leotime seed`; see `docs/10-operations.md` |
| Frontend feature split | Done | CRUD panels and `DashboardShell` under `apps/web/src/features/` |
| API error codes | Done | Structured JSON error payloads with validation `fields` |

All MVP slices in the original plan are complete. Next engineering slice: visual regression checks unless product scope changes.

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

Backlog engineering items such as visual regression checks unless product scope changes.
