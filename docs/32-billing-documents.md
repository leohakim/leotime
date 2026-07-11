# Billing Documents And Official PDFs

> **Status: partially implemented (2026-07-11).** Migration `000009_billing_documents.sql`, the `internal/billing` package, fiscal series, preview/issue/cancel/document routes, invoice UI, and document-aware backups are present. For the current HTTP contract, use [23-invoices-api.md](23-invoices-api.md). Issuance invariants and document atomicity were hardened in [H-INV-01](35-curated-hardening-backlog.md#h-inv-01--fiscal-issue-invariants-and-document-atomicity) (2026-07-11). Remaining restore safety work is tracked in [H-BACKUP-04](35-curated-hardening-backlog.md#h-backup-04--restore-database-and-documents-safely-together).

This document records the intended invoice document model and the parts already
delivered. It is not a legal-compliance claim. Current HTTP behavior is
documented in `docs/23-invoices-api.md`; known implementation limits are linked
from the curated hardening backlog.

## Goal

Generate the monthly or bimonthly settlement package the owner sends to a
client:

- official invoice PDF,
- Work Protocol PDF attached as appendix,
- configurable fiscal numbering,
- preview before issue,
- immutable issued files,
- durable backup and restore.

The design follows ADR `docs/adr/0004-billing-documents-official-pdfs.md`.

## Non-goals

- No claim of legal compliance for Spain, the United States, or any other
  jurisdiction.
- No electronic invoice format in the first delivery.
- No digital signature or certified fiscal software integration in the first
  delivery.
- No client portal or delivery email in the first delivery.
- No generic document builder beyond invoices and Work Protocols.

## Document Shape

The invoice PDF should preserve the sober structure of the current external
documents:

- Letter page size.
- Seller information at the top right.
- Large title on the left: `Invoice # <number>`.
- Issue date below the title.
- Client name, tax ID, address, city/state/country.
- Service description and project summary.
- Bordered amount table with description, hourly rate, quantity, adjustments,
  and total.
- Payment instructions block.
- Appendix reference: `Appendix: Work Protocol <number>`.

The Work Protocol PDF should:

- use the same seller block and number,
- show `Work Protocol # <number>`,
- show date and client name,
- render a bordered table with date, quantity, and tasks,
- support page breaks while keeping rows legible.

Real sample PDFs must not be committed to the repository.

## Domain Model

### Invoice draft

A draft invoice is editable and previewable. It can be created from billable
time entries for a client and period.

Drafts store:

- client ID,
- date range,
- selected fiscal series ID,
- issue date and due date,
- seller snapshot fields,
- client snapshot fields,
- billing description,
- project summary,
- tax and withholding inputs,
- adjustment rows such as volume discounts,
- Work Protocol detail level,
- notes.

Drafts do not own official numbers. The existing `invoice_number` field can
hold a non-official draft reference such as `DRAFT-inv_...` until issue. On
issue, the same field is replaced with the official fiscal number from the
selected series.

### Fiscal series

A fiscal series controls official numbering.

Example JSON shape:

```json
{
  "id": "ser_main",
  "code": "MAIN",
  "name": "Main invoices",
  "pattern": "{YYYY}-{SEQ:04}",
  "nextSequence": 9,
  "resetPolicy": "yearly",
  "active": true,
  "default": true
}
```

Supported placeholders:

| Placeholder | Meaning |
| --- | --- |
| `{YYYY}` | Issue year, four digits |
| `{YY}` | Issue year, two digits |
| `{SEQ}` | Sequence without padding |
| `{SEQ:04}` | Sequence padded to 4 digits |

The first delivery should support `never` and `yearly` reset policies. A series
can be inactive only when no draft uses it as default.

### Official package

Issuing creates an official package. The package contains:

- invoice number,
- fiscal series code and sequence,
- issue timestamp,
- frozen document snapshot JSON,
- `invoice.pdf` metadata,
- `work-protocol.pdf` metadata when enabled,
- render version,
- cancellation metadata when cancelled.

The package is immutable after issue. Cancellation records state; it does not
delete or regenerate files.

### Billing documents

Each generated file is represented in `billing_documents`:

| Field | Purpose |
| --- | --- |
| `id` | Stable document ID |
| `invoice_id` | Owning invoice |
| `kind` | `invoice_pdf` or `work_protocol_pdf` |
| `storage_path` | Relative path under the document root |
| `sha256` | Lowercase hex SHA-256 |
| `byte_size` | File size |
| `mime_type` | `application/pdf` |
| `render_version` | Template/renderer version |
| `created_at` | Creation time |

## Work Protocol Detail Levels

Detail is configured by client and overrideable by draft.

### `summary`

One row per day:

- date,
- total hours,
- project names.

Use when the client only needs a compact appendix.

### `standard`

One row per day:

- date,
- total hours,
- bullet list grouped by project/task,
- concise time entry descriptions.

This matches the sample Work Protocol style.

### `detailed`

One row per day:

- date,
- total hours,
- bullets with project, task, description, tags, and optional entry-level notes.

Use when the client needs more justification.

## Issue Flow

1. Owner creates a draft from a suggested or manual period.
2. Owner reviews invoice fields, adjustments, payment instructions, and Work
   Protocol detail.
3. Owner opens preview. Preview does not consume an official number.
4. Owner clicks issue.
5. Backend validates the draft.
6. Backend starts a database transaction.
7. Backend locks and increments the fiscal series.
8. Backend stores the official invoice number and status `issued`.
9. Backend freezes the document snapshot.
10. Backend renders PDFs to temporary files.
11. Backend verifies rendered PDFs and computes SHA-256 hashes from the temp
    render directory.
12. Backend marks the invoice issued and inserts `billing_documents` rows inside
    the transaction, then commits.
13. Backend promotes both PDFs into `/data/documents` only after commit.
14. If promotion fails, the backend reverts the issued state, deletes document
    rows, restores the fiscal sequence, and removes any partial files.

If rendering, metadata insertion, commit, or promotion fails, the invoice stays
`draft`, the fiscal sequence is unchanged, and no official file remains under
the document root.

## Period Suggestions And Warnings

The system should suggest likely periods, not enforce them.

Suggested periods:

- current month,
- previous month,
- current bimonthly range,
- previous bimonthly range,
- first and last uninvoiced billable entry for a client.

Warnings:

- selected period overlaps an existing issued or paid invoice,
- selected period contains no billable uninvoiced time,
- selected period contains running timers,
- selected period has billable entries without an hourly rate,
- selected period includes entries with overlap warnings.

Warnings do not block issue except for empty billable content and invalid
amounts.

## Storage

Document root defaults to `/data/documents` in Docker.

Configuration:

```text
LEOTIME_DOCUMENT_ROOT=/data/documents
```

Official file path format:

```text
invoices/<year>/<series>/<number>/invoice.pdf
invoices/<year>/<series>/<number>/work-protocol.pdf
```

The database stores the relative path. The API joins it with the configured
document root after validating that the path stays under the root.

## Backup And Restore

The S3 backup feature now includes document files in the same `.tar.gz`
delivery. Rollback-safe paired restore remains open in H-BACKUP-04.

Backup output is one archive object that contains:

```text
leotime.db
documents/
  invoices/
    ...
manifest.json
```

The manifest stores:

- database snapshot timestamp,
- document count,
- each document path,
- SHA-256 hash,
- byte size.

Restore validates:

- SQLite database opens and contains core tables,
- manifest is present,
- every document listed in `billing_documents` exists,
- hash and byte size match metadata,
- no restored path escapes the document root.

## Current HTTP API and follow-ups

All current routes require a valid session cookie. The authoritative current
route and legacy-status behavior is [23-invoices-api.md](23-invoices-api.md).

```text
GET    /api/v1/invoice-series
POST   /api/v1/invoice-series
PATCH  /api/v1/invoice-series/{seriesID}

POST   /api/v1/invoices/draft-from-time
GET    /api/v1/invoices/{invoiceID}
PATCH  /api/v1/invoices/{invoiceID}
POST   /api/v1/invoices/{invoiceID}/preview
POST   /api/v1/invoices/{invoiceID}/issue
POST   /api/v1/invoices/{invoiceID}/cancel
GET    /api/v1/invoices/{invoiceID}/documents
GET    /api/v1/invoices/{invoiceID}/documents/{documentID}/download
```

Period suggestions and client-level defaults remain follow-up work; they are
not part of the current route set.

### Issue response

```json
{
  "id": "inv_...",
  "status": "issued",
  "invoiceNumber": "2026-0009",
  "documents": [
    {
      "id": "doc_...",
      "kind": "invoice_pdf",
      "sha256": "b6f9...",
      "byteSize": 87554,
      "downloadUrl": "/api/v1/invoices/inv_.../documents/doc_.../download"
    },
    {
      "id": "doc_...",
      "kind": "work_protocol_pdf",
      "sha256": "24af...",
      "byteSize": 82392,
      "downloadUrl": "/api/v1/invoices/inv_.../documents/doc_.../download"
    }
  ]
}
```

## Current UI and follow-ups

The current invoice panel supports this workflow:

1. Select client.
2. Pick suggested period or custom period.
3. Choose fiscal series.
4. Review invoice details.
5. Choose Work Protocol detail.
6. Preview invoice and Work Protocol.
7. Issue official package.
8. Download invoice PDF, Work Protocol PDF, or both.

Follow-up client settings (not yet a current route):

- default fiscal series,
- default Work Protocol detail level,
- default service description,
- default payment instructions override if needed.

Follow-up profile/business defaults (not yet a current route):

- seller fiscal name,
- seller tax ID,
- seller address,
- seller email,
- default payment instructions,
- default fiscal series.

## Validation Rules and test coverage

The current issue service rejects:

- missing active fiscal series,
- invalid fiscal pattern,
- empty seller name,
- empty client name,
- no positive billable lines,
- negative tax rate,
- negative withholding,
- total below zero,
- renderer output that is not a PDF,
- hash mismatch after write,
- file path outside `LEOTIME_DOCUMENT_ROOT`.

The current issue service allows:

- periods that overlap existing invoices,
- multiple invoices for the same client and month,
- cancellation of issued invoices while preserving files,
- paid invoices to keep their documents downloadable.

## Testing Strategy

Existing backend tests cover:

- fiscal series formatting and sequence increment,
- transaction rollback does not consume numbers,
- issuing creates immutable documents,
- cancellation preserves number and files,
- Work Protocol detail levels produce expected rows,
- document path validation rejects traversal,
- SHA-256 metadata matches generated files,
- backup archive includes DB and documents,
- restore validates manifest and hashes.

Frontend tests:

- renders period suggestions,
- shows overlap warnings without blocking issue,
- preview flow opens invoice and Work Protocol tabs/panels,
- issue flow shows official number and document downloads,
- cancelled invoice keeps download buttons visible.

The next billing hardening smoke coverage should:

- issue one invoice from seed data,
- download both PDFs,
- run backup,
- restore into a fresh database,
- verify PDFs still download and hashes match.

## Rollout

1. Add schema and internal services behind existing invoice routes.
2. Keep old HTML/CSV/JSON export during migration.
3. Add official issue flow and document downloads.
4. Expand backup/restore to include documents.
5. Update docs and deploy guide.
6. Remove or de-emphasize old HTML export after the PDF path is stable.
