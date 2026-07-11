# H-DATA-02: Reports and Invoice Drafts Without Silent Truncation

**Status:** Done (2026-07-11)  
**Backlog:** [35-curated-hardening-backlog.md](../../35-curated-hardening-backlog.md#h-data-02--reports-and-invoice-drafts-without-silent-truncation)

## Problem

`ListTimeEntries` applies `LIMIT 500`. Reports and invoice draft selection reused it,
so totals and billable lines could silently exclude older records.

## Outcomes

- Interactive `GET /time-entries` stays capped at 500 and exposes `limit` + `truncated`.
- Reports query the full filtered range without a row cap.
- Invoice draft selection uses dedicated SQL for billable, uninvoiced entries without a cap.
- Store tests cover 501-entry report and draft scenarios.

## Gates

```bash
make test-api
make pre-commit
make smoke
```
