# ADR 0004: Billing Documents, Fiscal Series, and Official PDFs

## Status

Accepted.

## Implementation

**Not implemented** (2026-07-08). This ADR and [32-billing-documents.md](../32-billing-documents.md) describe the **next** invoice upgrade. Current production behavior remains [23-invoices-api.md](../23-invoices-api.md) (draft from time, HTML/CSV/JSON export, status workflow).

Planned migration: `000009_billing_documents.sql` (not in tree). Plan: [superpowers/plans/2026-07-08-billing-documents.md](../superpowers/plans/2026-07-08-billing-documents.md).

Do not implement ADR 0004 until hardening items in [34-known-gaps-and-audit.md](../34-known-gaps-and-audit.md) are triaged for restore safety and backup scope.

## Context

leotime already has invoice drafts created from billable time, frozen client
fields, status changes, and HTML/CSV/JSON export. That is enough for basic
tracking, but not enough for the owner's monthly or bimonthly settlement
workflow.

The owner needs the system to produce two documents that travel together:

- An official invoice PDF with seller details, client fiscal fields, service
  summary, amount table, discount or adjustment rows, payment instructions,
  and an appendix reference.
- A Work Protocol PDF that explains the intent of the worked hours by day,
  with configurable detail.

The sample documents are private business records and must not be committed.
They are used only to derive layout and workflow requirements: sober Letter
page layout, seller block at top right, title and date on the left, black
bordered tables, and no decorative branding.

The owner also needs the invoice book to behave like a real fiscal sequence:
official numbers are unique, configurable, and not reused. Draft previews must
not consume official numbers. Issued PDFs must remain available later and must
not change when time entries, client fields, profile fields, or templates
change.

## Decision

### Billing package

Add a billing document package around `invoices`, instead of replacing invoices
with a generic document system.

An invoice remains the core business entity. While it is a draft, it can be
edited and previewed. When it is issued, leotime creates an official package:

- assigns the next number from a configurable fiscal series,
- freezes all document input as a JSON snapshot,
- renders `invoice.pdf`,
- renders `work-protocol.pdf` when enabled,
- stores both files under `/data/documents`,
- stores document metadata, file paths, SHA-256 hashes, byte sizes, MIME type,
  and render version in SQLite,
- makes the issued package immutable.

The Work Protocol shares the invoice fiscal number. It is not a separate fiscal
sequence.

### Fiscal series

Fiscal series are configurable per owner. A series controls:

- code, such as `MAIN` or `CRAFTLINE`,
- display pattern, such as `{YYYY}-{SEQ:04}` or `INV-{YYYY}-{SEQ:04}`,
- next sequence number,
- padding and reset policy,
- whether the series is active,
- whether it is the default series for new drafts.

Production issue consumes a number inside the same database transaction that
marks the invoice as issued. Preview does not consume a number. Development and
test data can render preview numbers such as `DRAFT` or `PREVIEW-2026-0001`,
but those values are never stored as official invoice numbers.

### Multiple invoices per period

leotime will not reject overlapping client periods. The owner can emit as many
invoices as needed. The UI may suggest likely monthly or bimonthly periods and
warn when a chosen period overlaps an existing issued or paid invoice, but the
system must not block issuance solely because periods overlap.

Billable time entries already included in a non-cancelled invoice remain
excluded from automatic draft creation. Manual corrections can be handled by a
new draft or by cancellation plus a new issue.

### Issued document immutability

Issued PDFs are immutable. They cannot be edited, regenerated in place, or
deleted through normal product workflows.

If an issued invoice is wrong, the owner cancels it. Cancellation keeps the
official number occupied and keeps the PDFs downloadable. A replacement invoice
uses the next number in the fiscal series.

Formal corrective invoices are out of scope for this first delivery. They can
be added later as a new invoice type that references an original invoice.

### Work Protocol detail

Work Protocol detail is configurable at the client level and can be overridden
on a draft invoice before issue. Supported levels:

- `summary`: one row per day with total hours and project names.
- `standard`: one row per day with total hours and bullet items grouped by
  project/task, matching the sample document style.
- `detailed`: one row per day with more entry descriptions, task names, tags,
  and optional notes.

The chosen detail level and generated content are frozen at issue time.

### PDF generation and preview

Preview uses HTML rendered by the same document snapshot builder as official
PDFs. The preview may show a draft watermark or draft number. Official export
returns the persisted PDF files, not a browser print flow.

The first PDF renderer should be boring and local to the Go backend. It can use
HTML templates and a server-side renderer if that is the most reliable way to
match the sample layout, but the renderer must fit Docker-first deployment and
must be covered by smoke tests. The renderer interface must hide the concrete
engine so the implementation can move from HTML-to-PDF to a pure Go PDF library
later without changing handlers or store code.

### Storage and backup

Official PDFs are stored on disk under:

```text
/data/documents/invoices/<year>/<series>/<number>/invoice.pdf
/data/documents/invoices/<year>/<series>/<number>/work-protocol.pdf
```

SQLite stores metadata and hashes. The database does not store PDF blobs.

The S3 backup feature must be expanded in the same delivery to back up and
restore `/data/documents` together with the SQLite snapshot. A database restore
without matching official PDFs is not acceptable for production.

### Legal scope

This delivery builds a real internal invoice book with strong invariants:
unique fiscal numbering, immutable issued PDFs, audit metadata, and durable
backup. It does not claim legal compliance for any jurisdiction.

Electronic invoicing formats, digital signatures, certified fiscal software,
jurisdiction-specific tax rules, and formal corrective invoice workflows are
explicit follow-up work after the owner validates requirements with an advisor.

## Consequences

Good:

- Keeps leotime small and understandable.
- Uses the existing invoice workflow as the product center.
- Separates draft preview from official issue.
- Gives the owner stable PDFs for monthly and bimonthly settlements.
- Avoids inflating SQLite with binary data.
- Makes backup scope explicit and testable.

Tradeoffs:

- File storage adds restore and integrity checks that a DB-only design would
  avoid.
- PDF rendering adds operational surface to the Docker image.
- Overlap warnings are advisory; the owner remains responsible for choosing
  the correct billing period.
- Cancellation without formal corrective invoices is pragmatic, but may not be
  enough for every jurisdiction.

## Alternatives Considered

### Keep current invoices and add a PDF export

This is fast, but it keeps official numbering, immutable files, Work Protocols,
and backup scope underspecified. It does not meet the owner's goal of a real
invoice book.

### Store PDF blobs in SQLite

This simplifies backup because the database snapshot includes documents, but it
makes the database heavier, complicates inspection, and is less friendly for a
VPS owner who may want to browse or recover PDFs directly from a mounted data
directory.

### Generic document engine

A generic document/template engine could support more future document types,
but it adds abstraction before the product needs it. The first owner workflow
only needs invoices and Work Protocols.

### Block overlapping billing periods

This would prevent some mistakes, but it is too rigid. The owner may need
multiple invoices for the same client and period. leotime should warn and show
suggestions, not block valid business cases.

## Follow-up

- Formal corrective invoices.
- Jurisdiction-specific fiscal validation.
- Electronic invoice formats.
- Optional digital signatures or timestamping.
- Client-facing delivery email with attachments.
