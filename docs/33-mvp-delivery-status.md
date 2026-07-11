# MVP Delivery Status

**Status:** Single-owner MVP delivered; billing documents are partially implemented as of 2026-07-11.

The first deployable version of leotime is feature-complete for a single owner: time capture, reporting, invoicing, profile, email notifications, S3 backups, Solidtime import, and offline create/sync.

## Capability Matrix

| Area | Status | Notes |
| --- | --- | --- |
| Auth (login/logout/session) | Done | Bootstrap admin, HTTP-only cookie |
| Password reset email | Done | Outbox + reset UI |
| Clients CRUD | Done | Archive/restore, rates, multi-currency default |
| Projects CRUD | Done | Optional client, color, rate override |
| Tasks CRUD | Done | Optional project, billable default, inline rename |
| Tags CRUD | Done | Unique names, colors, archive |
| Manual time entries | Done | 1-minute precision, overlap warning |
| Timers | Done | Start/stop, edit start time, multiple open timers |
| Weekly timesheet | Done | Inline edit, week navigation |
| Calendar view | Done | Month grid, day selection |
| Reports | Done | Grouped totals, CSV/JSON export |
| Invoices | Done | Draft from time, fiscal series, preview, official PDFs, Work Protocol, document downloads, cancellation, and HTML/CSV/JSON export; issuance hardened in H-INV-01 |
| Dashboard | Done | Recent entries, heatmap, weekly bars, donut |
| Theme selector | Done | solid / light / dark / minimal |
| Profile settings | Done | Account, password, timezone, currency, timer mail toggles |
| Backup/restore email toggles | Done | Profile checkboxes, outbox templates |
| Offline queue MVP | Done | IndexedDB creates + timer/entry sync (see limitations in doc 34) |
| Still-running timer email | Done | In-process scheduler + SMTP/log |
| S3 daily backups + restore | Done | UI, CLI, scheduler, encrypted credentials |
| Solidtime ZIP import | Done | CLI + UI dry-run/import |
| Dev seed command | Done | `make seed` |
| CI pipeline | Done | Go tests, Vitest, Playwright, Docker build |
| Structured API errors | Done | `{ error: { code, message, fields? } }` |
| Frontend feature modules | Done | `apps/web/src/features/` |

## CLI Commands

| Command | Purpose |
| --- | --- |
| `leotime` (default) | Start HTTP server |
| `leotime seed` | Load demo data |
| `leotime import solidtime` | Import Solidtime ZIP |
| `leotime backup run \| list \| restore` | Backup operations |

See [Operations](10-operations.md).

## Database Migrations

Applied automatically on startup (embedded SQL):

| Migration | Topic |
| --- | --- |
| 000001 | Core schema (users, clients, projects, tasks, tags, time entries, invoices, app_settings) |
| 000002 | Solidtime import (`import_runs`, `external_mappings`) |
| 000003 | Tags archive support |
| 000004 | Profile columns on users / app_settings |
| 000005 | Email outbox + still-running timer columns |
| 000006 | Password reset tokens; outbox kind expansion |
| 000007 | Backup settings (S3 credentials) |
| 000008 | Backup/restore email notification toggles; outbox kinds |
| 000009 | Billing document metadata and invoice snapshots |
| 000010 | Invoice series and billing fields |
| 000011 | Billable defaults and rate backfill |

Migrations are embedded and applied automatically on startup. See the [curated hardening backlog](35-curated-hardening-backlog.md) for the missing upgrade-fixture coverage.

## Quality Gates (current)

```bash
make pre-commit   # fmt, vet, Go tests, Vitest, web build
make smoke        # HTTP health + login against running container
make test-e2e     # Playwright (CI)
```

## Explicitly Not in MVP

Documented in [Backlog](13-backlog.md) as **Later** or **Backlog**:

- Tauri desktop app
- Idle detection / activity tracking
- Full multi-device local-first sync
- Multi-user / team mode
- Public API tokens and webhooks
- Shared report links (nav placeholder only)
- Visual regression test suite
- **UI/UX experience themes** ([design spec](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md))

## Next Recommended Work

1. [H-BACKUP-04](35-curated-hardening-backlog.md#h-backup-04--restore-database-and-documents-safely-together) — restore database and documents safely together.
2. Complete the remaining P0/P1 slices in [35-curated-hardening-backlog.md](35-curated-hardening-backlog.md).
3. Revisit UI/UX experience themes only after hardening ([design spec](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md)).

After deploying to production, set `LEOTIME_ENV=production`, a strong `LEOTIME_BOOTSTRAP_PASSWORD`, `LEOTIME_METRICS_TOKEN`, and configure SMTP + S3 backup. See [VPS deployment](06-deploy-vps.md).
