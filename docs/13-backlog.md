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
| Next | Timer workflow | Start/stop, open timer, overlap awareness. |
| Backlog | Weekly timesheet | Editable weekly grid and grouped totals. |
| Backlog | Calendar view | Calendar inspection/editing of entries. |
| Backlog | Reports/export | CSV/JSON, grouped totals, optional timestamp hiding. |
| Backlog | Invoices | Draft/issued/paid/cancelled, PDF/export, multi-currency. |
| Backlog | Offline queue MVP | Create/edit offline and sync when online. |
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
| Backlog | Backup/restore command | Wrap SQLite backup and restore flows. |
| Backlog | CI pipeline | Run tests, build, docker build, and smoke in GitHub Actions. |

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
| Backlog | API reference index | One index linking each resource doc. |
| Backlog | Contributor tutorial | First issue walkthrough for Django/Python readers. |
