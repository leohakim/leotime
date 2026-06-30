# Product Vision

`leotime` is a personal time-tracking and invoicing system for one owner who wants the speed of a local tool and the convenience of a VPS-hosted web app.

It should feel closer to a fast workbench than to a heavy SaaS product.

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

## MVP Capabilities

- Login with a bootstrap admin user.
- Clients.
- Projects.
- Tasks.
- Tags.
- Time entries with one-minute precision.
- Overlap warnings, not overlap blocking.
- Calendar view.
- Weekly timesheet view.
- Rates by client and optionally by project.
- Multi-currency invoices.
- Invoice PDF/export.
- Report export as CSV and JSON.
- Spanish and English UI.
- Configurable layouts.
- Offline creation queue with sync when the app reconnects.

## Backlog Capabilities

The operational backlog lives in [`docs/13-backlog.md`](13-backlog.md). This section keeps the high-level product ideas.

- Tauri desktop app.
- Idle detection.
- Activity tracking.
- Full multi-device local-first sync.
- Import from Solidtime.
- Advanced invoice templates.
- More currencies and exchange-rate snapshots.
- Multi-user/team mode.
- Public API tokens.
- Webhooks.

## Non-Goals For The MVP

- Legal-grade tax compliance.
- Full accounting.
- Payroll.
- Enterprise roles and permissions.
- Real-time collaboration.
- Complex SaaS multi-tenancy.
