# Architecture Decision Records

ADRs capture durable decisions. Each record has a **status** (Accepted, etc.) and an **implementation** column that reflects what is in the codebase today—not what is planned.

| ADR | Title | Decision status | Implemented | Primary docs |
| --- | --- | --- | --- | --- |
| [0001](0001-stack-go-sqlite-react.md) | Go + SQLite + React | Accepted | **Yes** | [02-architecture-go.md](../02-architecture-go.md) |
| [0002](0002-in-process-scheduler-outbox.md) | In-process scheduler and email outbox | Accepted | **Yes** | [29-email-notifications.md](../29-email-notifications.md) |
| [0003](0003-s3-backup-encryption-and-restore.md) | S3 backups, encrypted credentials, restore | Accepted | **Yes** | [31-s3-daily-backups.md](../31-s3-daily-backups.md) |
| [0004](0004-billing-documents-official-pdfs.md) | Billing documents and official PDFs | Accepted | **No** | [32-billing-documents.md](../32-billing-documents.md) (planned) |

## Approved designs not yet ADRs

These specs are approved for planning but have **no ADR number** and **no production code** yet:

| Spec | Topic | Implementation |
| --- | --- | --- |
| [UI/UX experience themes](../superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md) | Presets, tokens, responsive shell | **Not started** |
| [Billing documents design](../superpowers/specs/2026-07-08-billing-documents-design.md) | Companion to ADR 0004 | **Not started** |

Implementation plans live under `docs/superpowers/plans/`. Do not treat them as shipped behavior until code and migrations land.

## When to add or update an ADR

Follow [docs/AGENTS.md](../AGENTS.md): when architecture or deployment expectations change, add or update an ADR and link the matching product/API doc.

After implementing an accepted ADR, update:

1. This table (`Implemented` column).
2. The ADR's **Implementation** section.
3. [MVP delivery status](../33-mvp-delivery-status.md) or [Backlog](../13-backlog.md) as appropriate.
