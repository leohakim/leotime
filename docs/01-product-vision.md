# Product Vision

`leotime` is a personal time-tracking and invoicing system for one owner who wants the speed of a local tool and the convenience of a VPS-hosted web app.

It should feel closer to a fast workbench than to a heavy SaaS product.

## MVP Status

**Delivered (2026-07-08).** The first deployable version is complete for a single owner. See [MVP delivery status](33-mvp-delivery-status.md) for the capability matrix and [Known gaps and audit](34-known-gaps-and-audit.md) for remaining hardening work.

## Primary User

The first product version is for one person:

- Tracks time every day.
- Uses clients, projects, tasks, and tags.
- Needs calendar and weekly timesheet views.
- Needs reports and invoices.
- Wants offline work to be possible.
- Wants a deployment that can be understood, backed up, and restored.

## Product Principles

- **Fast daily capture:** starting, stopping, editing, and correcting time must be low friction.
- **Trustworthy reporting:** reports should make overlaps visible without blocking valid real-world multitasking.
- **Simple deployment:** a VPS should only need Docker, a volume, environment variables, and a reverse proxy.
- **Local ownership:** SQLite makes the data inspectable, movable, and easy to back up.
- **Learning-friendly code:** important decisions live in docs, and code is organized for readers coming from Django.

## Delivered MVP Capabilities

- Login with a bootstrap admin user; password reset email.
- Clients, projects, tasks, tags.
- Time entries with one-minute precision; overlap warnings (not blocking).
- Timers (start/stop, edit start time).
- Calendar and weekly timesheet views.
- Hourly rates on clients and optional project override (`rates` table reserved for future rate history).
- Multi-currency invoice drafts with HTML/CSV/JSON export, fiscal series, preview, official PDFs, Work Protocol documents, and document-aware backups (with hardening tracked separately).
- Report export as CSV and JSON.
- Dashboard with heatmap, weekly bars, and project breakdown.
- Spanish and English UI; configurable layout modes and themes.
- Profile settings (timezone, currency, timer mail, backup mail toggles).
- Offline create queue with sync on reconnect (see limitations in doc 34).
- Still-running timer email (scheduler + outbox).
- S3 daily backups and in-app restore.
- Solidtime ZIP import.
- Dev seed command (`make seed`).

## Backlog Capabilities

The operational backlog lives in [`docs/13-backlog.md`](13-backlog.md). The next implementation queue is the [curated hardening backlog](35-curated-hardening-backlog.md). High-level ideas still outside MVP:

- Tauri desktop app.
- Idle detection and activity tracking.
- Full multi-device local-first sync.
- Multi-user/team mode.
- Public API tokens and webhooks.
- Shared report links.
- Advanced invoice templates and exchange-rate snapshots.

## Non-Goals For The MVP

- Legal-grade tax compliance.
- Full accounting or payroll.
- Enterprise roles and permissions.
- Real-time collaboration.
- Complex SaaS multi-tenancy.
