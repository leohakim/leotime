# Curated Hardening Backlog

> **Status:** active backlog, curated on 2026-07-11.
>
> This is the source of truth for the next implementation work. It reconciles
> the two static reviews from 2026-07-09 with the code and commits present on
> 2026-07-11. It supersedes their ordering, not the history recorded in
> [Known gaps and audit](34-known-gaps-and-audit.md).

## Operating rules

leotime is a Docker-first, single-owner time tracker and invoice tool. Make the
existing workflows trustworthy before expanding themes, routing, offline sync,
or team features.

An agent taking an item must:

1. Work on one slice only; do not combine a correctness or security fix with
   unrelated refactors.
2. Read the root and nested AGENTS.md instructions, this backlog, and the
   affected current-behavior document.
3. Create a dedicated implementation plan under docs/superpowers/plans before
   changing production behavior.
4. Use synthetic fixtures only. Never commit production exports, PDFs,
   credentials, reset tokens, or client data.
5. Update this item, docs/13-backlog.md, docs/34-known-gaps-and-audit.md, and
   the affected API or operations document in the same change.
6. Run make pre-commit before handoff, plus the item-specific gates below.

## Priority and release rule

Do not trust real fiscal invoices, production Solidtime imports, or production
restore until every P0 item is complete and its acceptance tests pass.

| Order | ID | Priority | Slice | Depends on |
| ---: | --- | --- | --- | --- |
| 1 | H-INV-01 | P0 | Fiscal issue invariants and document atomicity | none — **Done** |
| 2 | H-DATA-02 | P0 | Reports and invoice drafts without silent truncation | none |
| 3 | H-IMP-03 | P0 | Solidtime ZIP boundary and import privacy | none |
| 4 | H-BACKUP-04 | P0 | Restore database and documents safely together | H-INV-01 for document cases |
| 5 | H-PROD-05 | P1 | Production configuration and HTTP boundary safety | none |
| 6 | H-MIG-06 | P1 | Upgrade migration confidence | none |
| 7 | H-API-07 | P1 | JSON contract discipline and startup errors | none |
| 8 | H-UX-08 | P2 | Destructive-action clarity and focused maintenance | P0 complete |

P0 items are independent in code but should be delivered in the listed order.
P1 follows P0. P2 is intentionally opportunistic.

---

## H-INV-01 — Fiscal issue invariants and document atomicity

**Priority:** P0  
**Status:** Done (2026-07-11)

**Problem:** POST /api/v1/invoices/{id}/status accepts draft to issued. That
bypasses the fiscal series, frozen snapshot, and official PDF flow. The
official issue flow can also write final PDF files before a later database
failure, leaving untracked documents.

**Required outcome:**

- POST /issue is the only path from draft to issued.
- The legacy status route may mark issued invoices as paid only. It rejects
  issued, draft, and cancelled. Cancellation stays on POST /cancel with reason.
- Official number, status, frozen snapshot, document rows, and both PDFs become
  visible as one logical operation.
- A render, promotion, document-row insertion, or commit failure leaves the
  invoice draft and sequence unchanged, with no newly written official file.
- Download and export headers use safe generated filenames, never raw invoice
  text in header syntax.

**Expected files:**

- apps/api/internal/httpapi/invoices.go and invoice_billing.go
- apps/api/internal/store/invoice.go and invoice_documents.go
- apps/api/internal/billing/issue.go and storage.go
- router, store, and billing tests
- docs/23-invoices-api.md, docs/32-billing-documents.md, and ADR 0004

**Acceptance tests:**

1. A draft status request for issued is rejected with no document or sequence
   side effect.
2. A successful issue creates exactly two documents and an official number.
3. Forced failure after a file write and after file staging leaves no document
   under the document root and preserves the draft/series state.
4. Issued to paid and issued cancellation remain valid; all other shortcuts
   fail.
5. Series or invoice text cannot inject a malformed Content-Disposition header.

**Gates:** make test-api; make pre-commit; make smoke; make deploy-check.

---

## H-DATA-02 — Reports and invoice drafts without silent truncation

**Priority:** P0

**Problem:** ListTimeEntries ends with LIMIT 500. BuildTimeReport and the
invoice draft source query use it, so totals and billable lines can silently
exclude older records.

**Required outcome:**

- The interactive directory stays bounded and paginated.
- Reports and invoice draft selection use a dedicated unbounded source query.
- A report that includes raw timestamps either returns the full range or uses
  an explicit documented pagination contract; it never reports a total from a
  partial list.
- Every eligible billable, uninvoiced entry in the selected range is considered
  for a draft.

**Expected files:**

- apps/api/internal/store/time_entry.go, report.go, and invoice.go
- store and HTTP tests using 501 synthetic entries
- docs/18-time-entries-api.md, docs/22-reports-api.md, docs/23-invoices-api.md

**Acceptance tests:**

1. A 501-entry report has entryCount 501 and the exact summed duration.
2. A draft from 501 eligible entries includes every entry and exact totals.
3. The normal directory contract remains visibly limited or paginated.

**Gates:** make test-api; make pre-commit; make smoke.

---

## H-IMP-03 — Solidtime ZIP boundary and import privacy

**Priority:** P0

**Problem:** the request is capped at 32 MiB compressed, but the parser reads
every ZIP member fully into memory. Extra members are accepted and CLI imports
store the full local source path.

**Required outcome:**

- Permit exactly meta.json and the nine documented CSV files. Reject duplicate,
  unknown, absolute, and traversal-like names before parsing.
- Enforce at most 16 ZIP entries, 1 MiB for meta.json, 32 MiB per CSV, and
  128 MiB across all uncompressed members.
- Keep the 32 MiB compressed request limit.
- Store only a sanitized basename in import_runs.source_path.
- Preserve dry-run and idempotent external mappings.

**Expected files:**

- apps/api/internal/solidtimeimport/parser.go and importer.go
- apps/api/internal/httpapi/imports.go
- parser/importer and HTTP import tests
- docs/09-solidtime-import.md and docs/34-known-gaps-and-audit.md

**Acceptance tests:**

1. An oversized uncompressed CSV is rejected before import writes.
2. Unknown, duplicate, and traversal-like ZIP members are rejected.
3. A valid synthetic export still succeeds in dry-run and write modes.
4. A CLI run records only the ZIP basename.

**Gates:** make test-api; make bench; make import-solidtime-dry with a
synthetic export; make pre-commit.

---

## H-BACKUP-04 — Restore database and documents safely together

**Priority:** P0

**Problem:** restore replaces the database, then deletes the active document
root before copying archived files. A filesystem failure can leave restored
metadata pointing to missing or partial PDFs.

**Required outcome:**

- Validate database and archive manifest before changing live data.
- Copy restored documents to a sibling staging tree and verify it first.
- Keep pre-restore database and document tree until the new pair is active.
- If document promotion fails, restore both old database and old documents.
- A legacy database-only archive keeps the existing document tree untouched.

**Expected files:**

- apps/api/internal/backup/service.go and archive.go
- backup service and archive tests
- docs/31-s3-daily-backups.md, docs/06-deploy-vps.md, and ADR 0003

**Acceptance tests:**

1. Failed document promotion leaves original database and document hashes
   readable.
2. Successful archive restore makes document metadata match restored hashes.
3. Legacy database-only restore does not replace documents.
4. Maintenance mode remains active until paired restore succeeds.

**Gates:** make test-api; make pre-commit; make smoke; make deploy-check.

---

## H-PROD-05 — Production configuration and HTTP boundary safety

**Priority:** P1

**Problem:** invalid environment values silently choose defaults; log mail prints
reset tokens; auth limits trust a forwarding header; metrics accepts query
tokens; and default backup errors can expose internal details.

**Required outcome:**

- Invalid boolean, duration, and integer variables fail startup with their
  variable name.
- Production requires a non-default bootstrap password, secure cookies, and an
  explicit public base URL. Log mail is always redacted; production log mail
  requires an explicit opt-in if still supported.
- Direct peer address is the default rate-limit key. Forwarded headers require
  explicit trusted-proxy configuration.
- Metrics accepts constant-time Bearer credentials only, never query tokens.
- Backup responses are generic while diagnostic causes reach server logs with a
  request ID.
- Add X-Content-Type-Options, Referrer-Policy, and deny framing headers.

**Expected files:** config, mail/log, HTTP rate limit/security/backup/router
packages and tests; .env.example; docs/06-deploy-vps.md; docs/10-operations.md.

**Gates:** make test-api; make pre-commit; make docker-build; make smoke;
make deploy-check.

---

## H-MIG-06 — Upgrade migration confidence

**Priority:** P1

**Problem:** migration 000003 rebuilds tags inside the transactional migration
runner. No test starts from a version-2 database with time_entry_tags relations.

**Required outcome:**

- Start from a synthetic version-2 SQLite database with tags and tag links.
- Run Migrate and validate preserved links, foreign-key integrity, indexes, and
  migration versions through 000011.
- Keep migrations forward-only. Change the runner or add a migration only if
  this upgrade test proves the current behavior unsafe.

**Expected files:** apps/api/internal/db/migrate_test.go; migrate.go only if
the test proves it necessary; docs/03-data-model.md and docs/05-testing-strategy.md
if migration protocol changes.

**Gates:** make test-api; make pre-commit.

---

## H-API-07 — JSON contract discipline and startup errors

**Priority:** P1

**Problem:** mutation decoding accepts unknown fields and trailing JSON values,
and NewRouter panics when its document root cannot initialize.

**Required outcome:**

- Reject empty JSON, unknown fields, and a second JSON value while keeping the
  existing 1 MiB limit and structured envelope.
- Return an error from router construction; main reports it through the
  existing startup path.
- Audit every frontend payload before strict decoding is enabled.

**Expected files:** json_body.go, router.go, router/JSON tests, main.go; web API
client and form tests only where the payload audit finds a mismatch; docs/32-api-errors.md.

**Gates:** make test-api; make test-web; make pre-commit; make smoke.

---

## H-UX-08 — Destructive-action clarity and focused maintenance

**Priority:** P2

Add confirmation and precise copy for permanent deletion versus archive/restore.
Fold a small shared form or download helper into the same touched feature only
when it makes the change easier to test.

Do not start a generic CRUD framework, global CSS split, full Ui-to-features
move, or router rewrite in this slice.

**Gates:** make test-web; make pre-commit; make smoke.

---

## Explicitly deferred work

- UI/UX experience themes and responsive-shell redesign.
- Full offline update/archive/delete synchronization and multi-device conflict
  handling.
- Tauri, idle detection, activity tracking, public API tokens, webhooks,
  shared reports, and team features.
- Real routing, lazy feature loading, and global API/CRUD/CSS refactors.

Apply structural refactors only when a P0/P1 slice touches the same file and
the refactor makes its invariant clearer or easier to test.

## Removed as independent work

Do not restart these as standalone tickets: production login prefill, archived
tag assignment, timer startedAt, report date validation, session cleanup,
multiple-timer UI, preference hydration, legacy overview client, restore-path
exposure, backup field errors, draft invoice edit UI, and shared-report
placeholder navigation.

Invoice transitions are resolved for issuance: `POST /issue` is the only draft
to issued path; legacy `/status` accepts only `issued -> paid` (H-INV-01).

## Handoff checklist

- [ ] The item acceptance tests exist and pass.
- [ ] The affected current-behavior docs and item status are updated.
- [ ] No private data or secrets were added.
- [ ] make pre-commit passed.
- [ ] Every listed extra gate passed, or an unavailable local service is
      explicitly reported.

