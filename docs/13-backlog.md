# Backlog

This backlog is intentionally simple. It tracks product work before a dedicated issue tracker exists.

## Status Legend

- `Done`: implemented, tested, and committed.
- `Doing`: current active slice.
- `Next`: next likely slice.
- `Backlog`: planned but not started.
- `Later`: intentionally outside the MVP.

## Priority Phases

| Phase | Focus | Status |
| --- | --- | --- |
| **0** | Production hardening (restore safety, static files, metrics, bootstrap password, rate limits, JSON body limits) | **Done** |
| **1** | Backup stability (restore latest sort, validation, generic errors, HTTP tests, prune best-effort) | Next |
| **2** | UX/API coherence (`ApiError` everywhere, `taskProjectRequired`, offline queue, profile field errors) | Backlog |
| **3** | ADR 0004 billing documents (official PDFs, fiscal series, Work Protocol) | Backlog |
| **4** | Product polish (remaining audit medium/low items) | Backlog |
| **5** | UI/UX experience themes (10-sprint design spec) | Backlog |
| **6** | Tooling (visual regression, contributor tutorial) | Backlog |

See [Known gaps and audit](34-known-gaps-and-audit.md) for item IDs (C*, H*, M*, L*).

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

## Phase 0 — Production Hardening (Done)

| ID | Item | Notes |
| --- | --- | --- |
| C1 | Restore maintenance mode | Blocks API + scheduler during DB restore; UI reloads after success |
| H2 | Static file path traversal guard | `safeStaticFilePath` rejects paths outside `StaticDir` |
| H3 | Metrics auth | Hidden in production without `LEOTIME_METRICS_TOKEN`; Bearer or `?token=` when set |
| H4 | Production bootstrap password | `LEOTIME_ENV=production` requires explicit non-default `LEOTIME_BOOTSTRAP_PASSWORD` |
| M12 | Auth rate limits | Login 10/15min per IP; forgot-password 5/hour per IP+email |
| M14 | JSON body size limit | 1 MiB default on JSON handlers (`body_too_large`) |

## Phase 1 — Backup Stability (Next)

| ID | Item | Notes |
| --- | --- | --- |
| M1 | Restore `latest` sort | Sort S3 objects by `LastModified` before picking |
| M2 | Restore validation | `integrity_check` + migration version checks |
| M3 | Prune best-effort | Do not fail backup run after successful upload |
| M7 | Generic backup client errors | Do not leak S3 internals |
| M11 | Backup HTTP tests | Route coverage in `router_test.go` |
| M24 | Restore reload UX | Partially done via `requiresRestart`; verify cache clear |

## Phase 2 — UX / API Coherence (Backlog)

| ID | Item | Notes |
| --- | --- | --- |
| H5 | `ApiError` on all fetch paths | GET/DELETE/auth helpers |
| H6 | `taskProjectRequired` in UI | Tasks, timer, manual entry |
| H7 | Manual entry directory query | Not week-scoped only |
| H8 | Offline queue resilient flush | Continue independent ops; retry UI |
| H10 | Offline update/delete scope | Extend queue or document limitation |
| M17 | Profile `ApiError.fields` | Map field errors in UI |
| M22 | CRUD error states | Error pill in panels |

## Accepted ADRs and designs (not implemented)

| Status | Item | Notes |
| --- | --- | --- |
| Accepted, backlog | ADR 0004 billing documents | Official PDFs, fiscal series, Work Protocol; [32-billing-documents.md](32-billing-documents.md), [plan](superpowers/plans/2026-07-08-billing-documents.md). **Start after Phase 1.** |
| Approved, backlog | UI/UX experience themes | Six presets + SolidTime Exact; [design spec](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md) |

See [ADR index](adr/README.md) for implementation status of all records.

## Engineering Backlog

| Status | Item | Notes |
| --- | --- | --- |
| Done | Split frontend features | CRUD panels and dashboard shell under `apps/web/src/features/` |
| Done | API error codes | Structured `{ error: { code, message, fields } }` responses |
| Done | Seed/dev data command | `make seed` / `leotime seed` |
| Done | S3 backup/restore | Snapshot, S3 upload, scheduler, CLI, in-app restore |
| Done | CI pipeline | GitHub Actions: tests, build, Docker, smoke |
| Done | Phase 0 production hardening | Maintenance mode, metrics auth, rate limits, body limits |
| Backlog | Visual regression checks | Add screenshot checks after core UI stabilizes |
| Backlog | Contributor tutorial | First issue walkthrough for Django/Python readers |

## Documentation Backlog

| Status | Item | Notes |
| --- | --- | --- |
| Done | Product vision through MVP audit | See [00-documentation-index.md](00-documentation-index.md) |
| Done | Phase 0 env vars | `.env.example` (`LEOTIME_ENV`, `LEOTIME_METRICS_TOKEN`) |
| Backlog | Contributor tutorial | First issue walkthrough for Django/Python readers |
