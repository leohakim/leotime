# Billing Documents Design

## Problem

The current invoice feature can create a draft from billable time and export
HTML, CSV, or JSON. It does not yet behave like the owner's real settlement
workflow because it lacks official PDF persistence, configurable fiscal series,
Work Protocol appendices, immutable issued documents, and document-aware
backup/restore.

## Decision Summary

Build a billing document package around the existing invoice model. Drafts stay
editable and previewable. Issuing a draft creates an official package with a
fiscal number, frozen snapshot, invoice PDF, Work Protocol PDF, document hashes,
and immutable metadata.

## Requirements

- Fiscal series are configurable.
- Official numbers are assigned only on issue.
- Drafts may display a non-official draft reference.
- Preview never consumes a fiscal number.
- Multiple invoices for overlapping client periods are allowed.
- The UI may warn about overlapping periods and suggest likely billing ranges.
- Work Protocol detail is configured by client and overrideable by invoice.
- Supported detail levels are `summary`, `standard`, and `detailed`.
- Issued PDFs are immutable and remain downloadable after cancellation.
- Cancellation does not free or reuse the official number.
- Formal corrective invoices are outside the first delivery.
- Official PDFs are stored under `/data/documents`.
- SQLite stores document metadata and SHA-256 hashes, not PDF blobs.
- S3 backup and restore must include documents with the database.
- The delivery is an internal invoice book, not a legal compliance guarantee.

## Architecture

```text
HTTP handlers
  -> billing service
       -> store transaction
       -> snapshot builder
       -> renderer interface
       -> document file store
  -> SQLite metadata
  -> /data/documents PDFs
```

Key packages:

- `internal/store`: invoice, fiscal series, and document metadata persistence.
- `internal/billing`: issue orchestration and document snapshot construction.
- `internal/billing/render`: renderer interface and PDF implementation.
- `internal/billing/storage`: file writes, path validation, SHA-256 hashes.
- `internal/httpapi`: routes for series, preview, issue, cancel, and downloads.
- `apps/web/src/lib/invoiceUi.tsx`: draft, preview, issue, and downloads UI.

## Data Flow

### Draft

1. User chooses client and period.
2. Backend selects uninvoiced billable entries.
3. Backend creates editable invoice lines and Work Protocol source rows.
4. UI lets user choose fiscal series, detail level, and adjustments.

### Preview

1. UI requests preview for a draft.
2. Backend builds the same snapshot shape used for official PDFs.
3. Backend renders HTML preview with draft labels.
4. No fiscal number is consumed.

### Issue

1. Backend validates draft.
2. Backend opens one SQLite transaction.
3. Backend increments the selected fiscal series and assigns invoice number.
4. Backend freezes the snapshot.
5. Backend renders PDFs to temporary files.
6. Backend hashes and moves PDFs under `/data/documents`.
7. Backend inserts document metadata.
8. Backend marks invoice `issued`.
9. Backend commits.

If any step fails before commit, no official number is consumed and no issued
invoice is visible.

## Error Handling

- Invalid input returns HTTP `400`.
- Missing invoice or document returns HTTP `404`.
- Attempting to edit an issued invoice returns HTTP `409`.
- Document root write failures return HTTP `503`.
- Renderer failures return HTTP `500` and leave the draft editable.
- Backup/restore document integrity failures return HTTP `409` for restore and
  `500` for backup.

## Testing

The test suite must prove the invariants, not just happy paths:

- no sequence consumption on renderer failure,
- no file path traversal,
- immutable issued package,
- overlap warnings are advisory,
- cancellation preserves files,
- backup and restore keep PDFs and hashes intact.

## Open Follow-ups

These are not part of the first delivery:

- formal corrective invoice type,
- jurisdiction-specific tax validation,
- electronic invoice formats,
- digital signatures,
- client email delivery.
