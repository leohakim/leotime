# Documentation Index

Reading order for new contributors and operators.

## Start Here

| Doc | Purpose |
| --- | --- |
| [Product vision](01-product-vision.md) | Why leotime exists and MVP scope |
| [MVP delivery status](33-mvp-delivery-status.md) | What is shipped today (capability matrix) |
| [Known gaps and audit](34-known-gaps-and-audit.md) | Open bugs, limitations, and hardening backlog |
| [Curated hardening backlog](35-curated-hardening-backlog.md) | Current implementation queue with acceptance tests and gates |
| [Development workflow](08-development-workflow.md) | Day-to-day commands and mental model |
| [Contributor tutorial](40-contributor-tutorial.md) | First PR walkthrough for Django/Python readers (Spanish) |
| [Operations](10-operations.md) | Docker, backups, seed, metrics, smoke |
| [VPS deployment](06-deploy-vps.md) | Production deploy checklist |

## Architecture

| Doc | Purpose |
| --- | --- |
| [Go architecture](02-architecture-go.md) | Backend packages and request flow |
| [Data model](03-data-model.md) | SQLite tables and relationships |
| [Offline and sync](04-offline-sync.md) | IndexedDB queue and sync policy |
| [Testing strategy](05-testing-strategy.md) | Unit, integration, E2E, CI gates |
| [Solidtime import](09-solidtime-import.md) | ZIP export compatibility |
| [ADR index](adr/README.md) | All ADRs with implementation status |
| [ADR 0001: Stack](adr/0001-stack-go-sqlite-react.md) | Go + SQLite + React (implemented) |
| [ADR 0002: Scheduler](adr/0002-in-process-scheduler-outbox.md) | In-process mail scheduler (implemented) |
| [ADR 0003: S3 backups](adr/0003-s3-backup-encryption-and-restore.md) | Encrypted S3 backup/restore (implemented) |
| [ADR 0004: Billing PDFs](adr/0004-billing-documents-official-pdfs.md) | Official invoice PDFs (accepted, **partially implemented**) |

## HTTP API Reference

All authenticated routes use cookie sessions unless noted. Error envelope: [API error responses](32-api-errors.md).

| Resource | Doc |
| --- | --- |
| Clients | [11-clients-api.md](11-clients-api.md) |
| Projects | [14-projects-api.md](14-projects-api.md) |
| Tasks | [16-tasks-api.md](16-tasks-api.md) |
| Tags | [17-tags-api.md](17-tags-api.md) |
| Time entries | [18-time-entries-api.md](18-time-entries-api.md) |
| Timers | [19-timers-api.md](19-timers-api.md) |
| Reports | [22-reports-api.md](22-reports-api.md) |
| Invoices API | [23-invoices-api.md](23-invoices-api.md) (current) |
| Billing documents (current + hardening) | [32-billing-documents.md](32-billing-documents.md) + ADR 0004 |
| Dashboard stats | [24-dashboard-api.md](24-dashboard-api.md) |
| Profile settings | [26-profile-settings-api.md](26-profile-settings-api.md) |
| Email notifications | [29-email-notifications.md](29-email-notifications.md) |
| Password reset | [30-password-reset.md](30-password-reset.md) |
| S3 backups | [31-s3-daily-backups.md](31-s3-daily-backups.md) |

## Frontend Features

| Doc | Purpose |
| --- | --- |
| [Weekly timesheet](20-weekly-timesheet.md) | Timesheet grid UX |
| [Calendar view](21-calendar-view.md) | Monthly calendar UX |
| [Theme selector](25-theme-selector.md) | Palettes and persistence |
| [Experience presets](37-experience-presets.md) | Named preset catalog and SolidTime Exact reference |
| [UI/UX QA checklist](38-ui-ux-qa-checklist.md) | Responsive, visual, and accessibility gates after the UI/UX roadmap |
| [Visual regression](39-visual-regression.md) | Playwright PNG snapshot baselines and update workflow |
| [Solidtime-like theme](15-solidtime-theme.md) | Default visual language |
| [UI/UX visual audit](36-ui-ux-visual-audit.md) | Responsive baseline and prioritized friction map for the experience-theme roadmap |
| [Offline queue MVP](27-offline-queue-mvp.md) | Browser offline behavior |

## Planning

| Doc | Purpose |
| --- | --- |
| [Implementation plan](12-implementation-plan.md) | Delivery slices and quality gates |
| [Backlog](13-backlog.md) | Product and engineering backlog |
| [Curated hardening backlog](35-curated-hardening-backlog.md) | Agent-ready P0/P1/P2 queue |
| [ADR index](adr/README.md) | Architecture decisions and implementation status |
| [Billing documents](32-billing-documents.md) | Current billing behavior and remaining hardening |
| [UI/UX experience themes](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md) | Approved design (not shipped) |
| [Rust alternative plan](07-rust-axum-plan.md) | Future stack exploration |
