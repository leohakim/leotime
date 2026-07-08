# Backlog

This backlog is intentionally simple. It tracks product work before a dedicated issue tracker exists.

## Status Legend

- `Done`: implemented, tested, and committed.
- `Doing`: current active slice.
- `Next`: next likely slice.
- `Backlog`: planned but not started.
- `Later`: intentionally outside the MVP.

## Product Backlog

| Status | Item | Notes |
| --- | --- | --- |
| Done | Scaffold monorepo | Go API, React web, SQLite, Docker, docs. |
| Done | AI repo preparation | `AGENTS.md`, repo skills, project agents. |
| Done | Operational tooling | Make targets, Docker, metrics, Prometheus, Grafana, k6. |
| Done | Solidtime import foundation | CLI/service import, compatibility tables, tests, docs. |
| Done | Clients CRUD | Backend, API, UI, validation, tests, docs. |
| Done | Projects CRUD | Optional client assignment, color, hourly rate, archive. |
| Done | Tasks CRUD | Optional project assignment, billable default, archive. |
| Done | Tags CRUD | Unique names, colors, time-entry tagging. |
| Done | Manual time entries | One-minute precision, billable, overlap warning. |
| Done | Timer workflow | Start/stop, open timer, overlap awareness. |
| Done | Weekly timesheet | Editable weekly grid, week navigation, grouped totals. |
| Done | Calendar view | Monthly grid, day selection, inline editing. |
| Done | Reports/export | CSV/JSON, grouped totals, optional timestamp hiding. |
| Done | Invoices | Draft/issued/paid/cancelled, HTML/CSV/JSON export, multi-currency. |
| Done | Dashboard UI Solidtime compatibility | Recent entries, last 7 days, heatmap, weekly bars, billable totals, donut. |
| Done | Theme selector | Solidtime default, light, dark, minimal palettes with persistence. |
| Done | Profile Settings | Change password, email, name, timezone, currency, theme, etc. |
| Done | Offline queue MVP | Create/edit offline and sync when online. |
| Done | Still-running timer email | In-process scheduler, outbox, SMTP/log, docs. |
| Done | Timer notification settings UI | Threshold + toggle in profile settings. |
| Done | Password reset email | Outbox mail + login/reset UI. |
| Done | S3 daily backups + restore | UI, CLI, scheduler; 01:00 default, 365d retention; backup/restore email toggles in profile |
| Done | Backup/restore email notifications | Profile toggles + outbox; defaults: failure alerts on, success off |
| Later | Tauri desktop app | Desktop packaging after web MVP works. |
| Later | Idle detection | Helpful but not needed for first deployable MVP. |
| Later | Activity tracking | Backlog from original scope, not MVP. |
| Later | Full local-first sync | Multi-device conflict handling. |
| Later | Multi-user/team mode | Single owner first. |
| Later | Public API tokens | Useful after core API stabilizes. |
| Later | Webhooks | Useful after external integrations exist. |

## Engineering Backlog

| Status | Item | Notes |
| --- | --- | --- |
| Backlog | Split frontend features | Move large dashboard sections into feature components. |
| Backlog | Visual regression checks | Add screenshot checks after core UI stabilizes. |
| Backlog | API error codes | Move from plain error text to structured validation errors. |
| Backlog | Seed/dev data command | Make UI development easier without manual entry. |
| Done | S3 backup/restore | Snapshot, S3 upload, scheduler, CLI, in-app restore; see `docs/31-s3-daily-backups.md` |
| Done | CI pipeline | GitHub Actions: tests, build, Docker, smoke |

## Documentation Backlog

| Status | Item | Notes |
| --- | --- | --- |
| Done | Product vision | `docs/01-product-vision.md`. |
| Done | Architecture docs | Go, data model, offline, testing, deploy, operations. |
| Done | Rust alternative plan | `docs/07-rust-axum-plan.md`. |
| Done | Implementation plan | `docs/12-implementation-plan.md`. |
| Done | Backlog | This file. |
| Done | Projects API | `docs/14-projects-api.md`. |
| Done | Tasks API | `docs/16-tasks-api.md`. |
| Done | Tags API | `docs/17-tags-api.md`. |
| Done | Time entries API | `docs/18-time-entries-api.md`. |
| Done | Timers API | `docs/19-timers-api.md`. |
| Done | Invoices API | `docs/23-invoices-api.md`. |
| Done | Dashboard API | `docs/24-dashboard-api.md`. |
| Done | Theme selector | `docs/25-theme-selector.md`. |
| Done | Profile settings | `docs/26-profile-settings-api.md`. |
| Done | Offline queue MVP | `docs/27-offline-queue-mvp.md`. |
| Done | Email notifications | `docs/29-email-notifications.md`. |
| Done | Password reset | `docs/30-password-reset.md`. |
| Done | S3 daily backups | `docs/31-s3-daily-backups.md`. |
| Backlog | API reference index | One index linking each resource doc. |
| Backlog | Contributor tutorial | First issue walkthrough for Django/Python readers. |
