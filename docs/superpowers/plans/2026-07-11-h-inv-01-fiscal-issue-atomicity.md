# H-INV-01: Fiscal Issue Invariants and Document Atomicity

**Status:** Done (2026-07-11)  
**Backlog:** [35-curated-hardening-backlog.md](../../35-curated-hardening-backlog.md#h-inv-01--fiscal-issue-invariants-and-document-atomicity)

## Problem

1. `POST /api/v1/invoices/{id}/status` accepts `draft -> issued`, bypassing fiscal
   series, snapshot, and official PDFs.
2. `IssueService.Issue` writes final PDFs under `LEOTIME_DOCUMENT_ROOT` before the
   database transaction commits, so a later DB failure can orphan files.

## Outcomes

- `POST /issue` is the only path from `draft` to `issued`.
- `POST /status` accepts only `issued -> paid`; rejects `issued`, `draft`, and
  `cancelled` targets.
- Official number, status, snapshot, document rows, and both PDFs become visible
  together: commit DB metadata first, then promote rendered PDFs from a temp
  directory; revert DB and remove any partial files if promotion fails.
- Download and export headers use sanitized filenames.

## Implementation

### Store

- Tighten `canTransitionInvoiceStatus` to allow only `issued -> paid` (and no-op
  same status).
- Add `RevertInvoiceIssueTx` to restore draft state, delete billing documents,
  and roll back the fiscal series counter when file promotion fails after commit.

### Billing

- Reorder `IssueService.Issue`: render to temp dir, hash sources, DB transaction
  (number, issued mark, document rows), commit, then `WriteOfficial` for both
  PDFs; on promotion failure call revert and `RemoveOfficial` for partial files.
- Add `HashSourceFile`, `SafeDownloadFilename`, and `RemoveOfficial` helpers.

### HTTP

- Use `SafeDownloadFilename` in document download and invoice export handlers.
- Update router test: `draft -> issued` via `/status` returns 400.

### Tests

1. Draft status request for `issued` rejected with no side effects.
2. Successful issue creates exactly two documents and official number.
3. Forced failure after document insert and after first file write leaves no
   files under document root and preserves draft/series state.
4. `issued -> paid` still works; other shortcuts fail.
5. Malformed invoice text cannot inject `Content-Disposition` syntax.

## Gates

```bash
make test-api
make pre-commit
make smoke
make deploy-check
```
