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

All MVP slices in the original plan are complete.

## Post-MVP Documentation

- [Documentation index](00-documentation-index.md)
- [MVP delivery status](33-mvp-delivery-status.md)
- [Known gaps and audit](34-known-gaps-and-audit.md)
- [Curated hardening backlog](35-curated-hardening-backlog.md)
- [ADR index](adr/README.md)

## Accepted roadmap

| Item | ADR / spec | Current behavior |
| --- | --- | --- |
| UI/UX experience themes | [Design spec](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md) | Sprint 3 selector [complete](25-theme-selector.md); shell/nav redesign in Sprint 4+ |

## Recently delivered

| Item | ADR / spec | Notes |
| --- | --- | --- |
| Billing documents + official PDFs | [ADR 0004](adr/0004-billing-documents-official-pdfs.md), [32-billing-documents.md](32-billing-documents.md) | [23-invoices-api.md](23-invoices-api.md); document-aware backups in [31-s3-daily-backups.md](31-s3-daily-backups.md) |
| UI/UX Sprint 1 visual audit | [Experience design](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md) | [24 responsive baselines and prioritized findings](36-ui-ux-visual-audit.md) |
| UI/UX Sprint 3 experience selector | [Experience design](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md) | [Preset/nav controls and local persistence](25-theme-selector.md) |

Implementation plans: `docs/superpowers/plans/`.

## Current Next Task

Start **UI/UX Sprint 4: shell and navigation** from the
[approved design](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md),
building on the [Sprint 3 selector](25-theme-selector.md). Limit the slice to
responsive shell components (`SidebarNav`, compact sidebar, topbar, bottom nav)
without breaking hash routes, timer access, settings, language, offline status,
or logout.

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
