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
| **5** | UI/UX experience themes (10-sprint design spec) | **Done** |
| **6** | Tooling (visual regression, contributor tutorial) | **Done** |
| **7** | Curated hardening (billing, data, import, restore, production, UX) | **Done** |
| **8** | Daily workflow (Slack standup summary from time entries) | **Done** |
| **9** | AI-enriched daily summary (edit → approve workflow + local enricher) | **In progress** — [design spec](superpowers/specs/2026-07-12-ai-daily-summary-enrichment-design.md) |
| **10** | VCS context providers for AI summaries (GitHub, Gitea, GitLab) | **Backlog** — server-side, read-only context for the daily-summary prompt; includes GitHub.com/Enterprise, self-hosted Gitea such as Osoigo, and GitLab.com/self-hosted GitLab. |

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
| Backlog | Client volume discounts | Per-client rule: when monthly billable hours exceed a threshold, apply a % discount to the hourly rate (e.g. >100h → 10% off). Must affect invoice line pricing for that period. |
| Backlog | Fixed-hour monthly plans | Named retainer-style plans with fixed hours and fixed price (e.g. "Plan de mantenimiento preventivo, 15h, $750"), assignable to clients; track hours consumed vs included. |
| Backlog | Payment method profiles | Multiple bank/transfer detail sets beyond the single `payment_instructions` text (e.g. IBAN for EU clients, ACH for USA). Selectable per client or invoice so PDFs show the right transfer data. |
| Backlog | Historical document archive | Upload and keep pre-leotime (or external) invoice and Work Protocol PDFs as a searchable archive of record; each document must be linked to an existing client. Stored under the document root and included in S3 backups. See design notes below. |
| Backlog | VCS context providers for AI summaries | Connect GitHub, self-hosted Gitea (including Osoigo), and GitLab so a scoped daily summary can use its real development activity. See detailed intent below. |
| Later | Tauri desktop app | Desktop packaging after web MVP works. |
| Later | Idle detection | Helpful but not needed for first deployable MVP. |
| Later | Activity tracking | Backlog from original scope, not MVP. |
| Later | Full local-first sync | Multi-device conflict handling. |
| Later | Multi-user/team mode | Single owner first. |
| Later | Public API tokens | Useful after core API stabilizes. |
| Later | Webhooks | Useful after external integrations exist. |

### Historical document archive (design intent)

Goal: make leotime the owner's durable work and accounting archive, not only the
system that issues new documents. When adoption starts mid-series (e.g. next
issued number is `A 0003 00000027`), the owner can still keep earlier invoices
and Work Protocols on hand inside the app and inside encrypted S3 backups,
alongside personal copies (e.g. iCloud).

In scope:

- Upload external PDFs already sent from email or a previous tool (invoices and
  Work Protocols first; other document kinds only if needed later).
- **Required client link:** every archived PDF must belong to a client already
  loaded in leotime. Reject uploads without a valid `clientId`; filter and
  browse by client. Create the client first if the historical counterparty is
  new to the platform.
- Free-form display number and optional series/label so foreign numbering is
  allowed (`A 0003 00000001` … `00000026`, or a completely different scheme).
- Light metadata only: type, display number, issue/service date, required
  client, currency/amount when known, notes, original filename, SHA-256.
- Browse/filter/download in the UI; files live under `LEOTIME_DOCUMENT_ROOT`
  and ride the existing document-aware backup/restore path.
- Clear separation from fiscal issuance: archived uploads must **not** consume
  or rewrite the live fiscal series counter used by `issue`.

Out of scope for the first slice:

- Re-entering historical invoices as structured time-entry / line-item data.
- OCR, email import, or automatic reconstruction of billable hours from PDFs.
- Orphan / unassigned archive documents (no client).
- Claiming legal or tax compliance; this is an operational archive of PDFs the
  owner already issued elsewhere.

### VCS context providers for AI summaries (design intent)

Goal: give the daily-summary enricher a trustworthy, provider-neutral view of
the work that actually happened in every repository used by the owner. The AI
receives this context together with leotime time entries, the optional local
Cursor context, and the user's manual note, so it can write a grounded Slack
standup rather than infer work from hours alone.

Initial providers:

- **GitHub:** GitHub.com and GitHub Enterprise Server.
- **Gitea:** self-hosted instances, including the Osoigo installation. The
  configured base URL is part of the connection, not a hard-coded domain.
- **GitLab:** GitLab.com and self-hosted instances.

In scope:

- A single VCS connection model with provider kind, base URL, account identity,
  encrypted read-only token, connection health, and an optional **default
  client**. This lets a dedicated Gitea/GitLab service belong directly to one
  client while still allowing a shared service to host repositories for several
  clients.
- An explicit repository link with a required effective client and an optional
  leotime project. A repository either inherits the connection's default client
  or names its own client; it must resolve to exactly one client before it can
  contribute VCS context. Different repositories on the same Gitea instance
  may therefore serve different clients without mixing their activity.
- A normalized context bundle for a requested day and client/project scope:
  commits (hash, subject, author, timestamp, branch when available, changed-file
  names and aggregate stats), pull/merge requests (title, state, author,
  reviewers, labels, linked commits, and merge result), code-review activity,
  and issue/ticket references exposed by the provider.
- A deterministic collector that begins with the requested client, then filters
  by the configured owner identity, that client's linked service/repositories,
  optional project, and selected date. It must deduplicate the same commit or
  review seen through more than one API and reject a repository without an
  unambiguous client relationship.
- A compact, labelled VCS section inserted into the existing AI prompt. The
  generated text must say only what the collected facts support and keep the
  current editable draft → approval workflow.
- UI to create, test, rotate, disable, and remove connections; link them to
  projects; select whether commits, reviews, merge requests, and issue
  references contribute to a summary; and preview the exact normalized facts
  before enrichment.
- Cache and rate-limit-aware fetching so repeated editor previews do not create
  unnecessary provider traffic. A transient provider failure must leave the
  template/local-enricher path usable and show an honest partial-context state.

Security and privacy requirements:

- Tokens are encrypted at rest, never returned by the API, masked in the UI,
  and never logged. Connections request the smallest provider-specific,
  read-only scope; no repository write, clone, webhook, or administrative
  permission is needed.
- The server makes outbound requests only to validated configured provider URLs.
  Self-hosted URLs need strict scheme/host validation and an explicit allowlist
  policy so a connection cannot become an SSRF path into unrelated internal
  services.
- The prompt gets capped, redacted facts—not raw diffs, full source files,
  secrets, or unbounded review discussions. The preview makes the exact payload
  inspectable before spending Cursor credits.

Delivery order:

1. Define the provider-neutral connection, client-default and repository-level
   client links, optional project link, encrypted credentials, repository
   identity, and normalized VCS context contract; deliver the GitHub adapter as
   the reference implementation.
2. Add the Gitea adapter and validate it against the Osoigo instance, including
   a testable self-hosted base URL and read-only token setup.
3. Add the GitLab adapter for GitLab.com and self-hosted GitLab through the same
   contract.
4. Add connection management, context preview, caching/rate-limit handling,
   failure telemetry, synthetic provider fixtures, and end-to-end daily-summary
   tests.

Out of scope for the first delivery:

- Writing commits, comments, approvals, issues, webhooks, or changing repository
  settings from leotime.
- Cloning repositories or sending raw source code/diffs to the AI provider.
- Broad generic Git support without a provider API; a plain remote URL alone
  cannot reliably provide pull/merge-request or review context.
- New providers such as Bitbucket or Azure DevOps. They can be added later as
  adapters once the three initial providers prove the contract.

Acceptance criteria:

1. A client-scoped summary includes only normalized VCS activity from that
   client's linked GitHub, Gitea, or GitLab services and repositories for the
   selected day.
2. A project-scoped summary narrows that client's context to its linked
   repositories. A summary covering multiple projects combines their activity
   without mixing repositories, clients, or duplicate commits.
3. Missing credentials, invalid hosts, rate limits, or provider outages produce
   a clear partial-context result and never block drafting, editing, approval,
   or the local Git/Cursor path.
4. Tokens, raw diffs, and source contents do not appear in API responses, logs,
   persisted summary text, or AI prompts by default.

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
| Done | UI/UX experience themes | Ten-sprint roadmap complete; see [QA checklist](38-ui-ux-qa-checklist.md) |

See [ADR index](adr/README.md) for implementation status of all records.

### UI/UX experience roadmap

| Sprint | Status | Outcome |
| ---: | --- | --- |
| 1 | **Done** | [Responsive visual audit and prioritized `UXA-*` findings](36-ui-ux-visual-audit.md) |
| 2 | **Done** | Root experience attributes, semantic token foundation, legacy preference compatibility, and `custom` state |
| 3 | **Done** | Experience selector, preset catalog, nav modes, and local persistence |
| 4 | **Done** | Responsive shell, compact sidebar, bottom navigation, and UXA-001 fix |
| 5 | **Done** | Timer capture bar, manual-entry scroll/focus, sticky editor, form-first stacking |
| 6 | **Done** | Compact timesheet summary rows, expand-to-edit, UXA-004 fix |
| 7 | **Done** | Tablet dashboard stacking, calendar toolbar layout, UXA-003 fix |
| 8 | **Done** | Report/invoice workbenches, auto-loaded preview, UXA-008 fix |
| 9 | **Done** | Preset pack polish, login hero, SolidTime Exact reference, UXA-009 fix |
| 10 | **Done** | Shared feedback states, audit tooling, QA checklist, UXA-010 fix |

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
| Done | Visual regression checks | `make audit-ui-regression` + committed PNG baselines |
| Done | Contributor tutorial | [First-issue walkthrough for Django/Python readers](40-contributor-tutorial.md) |
| Done | Daily Slack summary | Narrative `GET /api/v1/reports/daily-summary` + `#daily-summary` UI with copy |
| In progress | AI-enriched daily summary | Draft/approve workflow, scoped summaries, local enricher, AI settings UI, Cursor Cloud API call from enricher — [spec](superpowers/specs/2026-07-12-ai-daily-summary-enrichment-design.md) |

## Documentation Backlog

| Status | Item | Notes |
| --- | --- | --- |
| Done | Product vision through MVP audit | See [00-documentation-index.md](00-documentation-index.md) and [35-curated-hardening-backlog.md](35-curated-hardening-backlog.md) |
| Done | Contributor tutorial | [40-contributor-tutorial.md](40-contributor-tutorial.md) |
| Done | Phase 0 env vars | `.env.example` (`LEOTIME_ENV`, `LEOTIME_METRICS_TOKEN`) |
