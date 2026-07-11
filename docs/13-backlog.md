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
| **1** | Backup stability (restore latest sort, validation, generic errors, HTTP tests, prune best-effort) | **Done** |
| **2** | UX/API coherence (`ApiError` everywhere, `taskProjectRequired`, offline queue, profile field errors) | **Done** |
| **3** | ADR 0004 billing documents (official PDFs, fiscal series, Work Protocol) | **Done** |
| **4** | Product polish (remaining audit medium/low items) | **Done** |
| **5** | UI/UX experience themes (10-sprint design spec) | **Doing** |
| **6** | Tooling (visual regression, contributor tutorial) | Backlog |
| **7** | Curated hardening (billing, data, import, restore, production, UX) | **Done** |

See the [curated hardening backlog](35-curated-hardening-backlog.md) for the current H-* queue. The IDs in [Known gaps and audit](34-known-gaps-and-audit.md) are historical findings and fix records.

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
| Done | Invoices | Draft/issued/paid/cancelled, fiscal series, official PDFs, Work Protocol, document downloads, HTML/CSV/JSON export, multi-currency. |
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

## Phase 1 — Backup Stability (Done)

| ID | Item | Notes |
| --- | --- | --- |
| M1 | Restore `latest` sort | Newest S3 object by `LastModified` |
| M2 | Restore validation | `PRAGMA integrity_check` + `schema_migrations` version |
| M3 | Prune best-effort | Upload success kept when retention delete fails |
| M7 | Generic backup client errors | `backup_remote_storage_failed`; no S3 internals in API |
| M11 | Backup HTTP tests | Auth, confirm, secrets key, status, generic remote errors |
| M24 | Restore reload UX | Done in Phase 0 via `requiresRestart` |

## Phase 2 — UX / API Coherence (Done)

| ID | Item | Notes |
| --- | --- | --- |
| H5 | `ApiError` on all fetch paths | `apiGet`/`apiDelete`/`apiPost` helpers; auth and CRUD migrated |
| H6 | `taskProjectRequired` in UI | Tasks, timer, manual entry, inline timesheet |
| H8 | Offline queue resilient flush | Continue independent ops after failure |
| H10 | Offline update/delete scope | Documented in UI (`offlineCreatesOnly`) |
| M17 | Profile `ApiError.fields` | Map field errors in profile and password forms |
| M22 | CRUD error states | `QueryErrorBanner` + retry in shell panels |

## Phase 4 — Product Polish (Done)

| ID | Item | Notes |
| --- | --- | --- |
| H7 | Manual entry directory query | **Done** — 90-day dedicated query, honest count, paginated load more |
| M5 | Archived tags on time entries | **Done** — reject archived tag IDs in store validation |
| M15 | Report date validation | **Done** — RFC3339 + range checks in reports API |
| M18 | Report export gating | **Done** — disable CSV/JSON until preview succeeds |
| M25 | Invoice local client filter | **Done** — hide offline `local_*` clients in invoice draft picker |
| M4 | Invoice status transitions | **Done** — allow draft→issued and issued→paid only |
| M13 | Session lookup failures | **Done** — return 503 instead of masking as unauthenticated |
| M16 | Dashboard timer restart offline | **Done** — queue restart via offline `startTimer` |
| M6 | Timer start honors `startedAt` | **Done** — optional RFC3339 start time on `StartTimer` |
| M8 | Backup resolve field errors | **Done** — structured validation errors from S3 config resolve |
| M20 | Reports nav and cache keys | **Done** — rename to Informes/reporting, drop dead `fetchOverview` |
| M23 | Profile preference hydration | **Done** — sync locale/layout/theme from profile on login |
| M10 | Outbox duplicate send guard | **Done** — quarantine row when mark sent fails after delivery |
| L2 | Timer `ErrInvalidTimerInput` | **Done** — use for `startedAt` validation on start/update |
| L3 | Backup schedule hour field name | **Done** — validation errors use `scheduleHour` |
| L5 | Restore safety path in API | **Done** — omit `safetySnapshotPath` from JSON response |
| L6 | Shared reports nav placeholder | **Done** — hide nav until implemented |
| L8 | Auth dev credentials in prod | **Done** — empty login defaults outside dev |
| L9 | Import summary i18n | **Done** — `importEntitySeen` translation key |
| L10 | Decorative select-all checkbox | **Done** — removed from timesheet toolbar |
| L11 | Offline 502/503 detection | **Done** — queue mutations on gateway/service errors |
| L1 | Auth artifact cleanup | **Done** — scheduler purges expired sessions and reset tokens |
| L4 | JSON encode error logging | **Done** — `writeJSON` logs encoder failures |
| L7 | Invoice draft edit UI | **Done** — PATCH draft fields from invoice detail |
| M21 | Multiple open timers UX | **Done** — warning banner and stop controls for extras |
| M9 | `rates` table scope | **Accepted** — reserved for future rate history per product vision |

## Accepted ADRs and designs

| Status | Item | Notes |
| --- | --- | --- |
| Accepted, partially implemented | ADR 0004 billing documents | Official PDFs, fiscal series, Work Protocol, document-aware backups, H-INV-01 issuance hardening, and H-BACKUP-04 rollback-safe restore exist. |
| Approved, doing | UI/UX experience themes | Sprint 4 shell complete; timer/quick capture next in the [design spec](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md) |

See [ADR index](adr/README.md) for implementation status of all records.

### UI/UX experience roadmap

| Sprint | Status | Outcome |
| ---: | --- | --- |
| 1 | **Done** | [Responsive visual audit and prioritized `UXA-*` findings](36-ui-ux-visual-audit.md) |
| 2 | **Done** | Root experience attributes, semantic token foundation, legacy preference compatibility, and `custom` state |
| 3 | **Done** | Experience selector, preset catalog, nav modes, and local persistence |
| 4 | **Done** | Responsive shell, compact sidebar, bottom navigation, and UXA-001 fix |
| 5 | **Next** | Timer and quick capture |
| 6 | Backlog | Weekly timesheet |
| 7 | Backlog | Calendar and dashboard |
| 8 | Backlog | Reports and invoices |
| 9 | Backlog | Initial preset pack |
| 10 | Backlog | Visual QA, accessibility checks, and documentation |

## Engineering Backlog

| Status | Item | Notes |
| --- | --- | --- |
| Done | Split frontend features | CRUD panels and dashboard shell under `apps/web/src/features/` |
| Done | API error codes | Structured `{ error: { code, message, fields } }` responses |
| Done | Seed/dev data command | `make seed` / `leotime seed` |
| Done | S3 backup/restore | Snapshot, S3 upload, scheduler, CLI, in-app restore |
| Done | CI pipeline | GitHub Actions: tests, build, Docker, smoke |
| Done | Phase 2 UX/API coherence | ApiError helpers, taskProjectRequired, offline flush, profile fields, query error banners |
| Done | Curated hardening | H-INV-01 through H-UX-08 in [35-curated-hardening-backlog.md](35-curated-hardening-backlog.md) |
| Backlog | Visual regression checks | Add screenshot checks after core UI stabilizes |
| Backlog | Contributor tutorial | First issue walkthrough for Django/Python readers |

## Documentation Backlog

| Status | Item | Notes |
| --- | --- | --- |
| Done | Product vision through MVP audit | See [00-documentation-index.md](00-documentation-index.md) and [35-curated-hardening-backlog.md](35-curated-hardening-backlog.md) |
| Done | Phase 0 env vars | `.env.example` (`LEOTIME_ENV`, `LEOTIME_METRICS_TOKEN`) |
| Backlog | Contributor tutorial | First issue walkthrough for Django/Python readers |
