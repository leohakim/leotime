# Billing Documents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add official invoice and Work Protocol PDFs with configurable fiscal series, preview before issue, immutable issued documents, and document-aware backup/restore.

**Architecture:** Keep `invoices` as the product center and add a billing document package around it. Drafts are editable and previewable; issue assigns the next fiscal number, freezes a snapshot, renders PDFs, writes files under `/data/documents`, stores SHA-256 metadata in SQLite, and makes the package immutable.

**Tech Stack:** Go 1.26, SQLite, React/Vite/TypeScript, existing S3 backup package, server-side PDF rendering behind a Go interface, Vitest, Go integration tests, Playwright smoke coverage where needed.

**Spec:** `docs/32-billing-documents.md`, ADR `docs/adr/0004-billing-documents-official-pdfs.md`

## Global Constraints

- Do not commit real client documents, real Solidtime export ZIP files, bank data, or personal production data.
- Keep Docker-first deployment working.
- Keep documentation updated whenever behavior, commands, schema, import mapping, or deployment expectations change.
- Prefer explicit Go and SQL over hidden framework magic.
- Prefer focused React components and stable layouts over decorative UI.
- Official numbers are assigned only on issue, never on preview.
- Drafts may use `invoice_number` for a non-official draft reference until issue.
- Issued PDFs are immutable and remain downloadable after cancellation.
- Multiple invoices for overlapping client periods are allowed.
- PDFs are stored under `/data/documents`; SQLite stores metadata and SHA-256 hashes.
- Backup and restore must include both SQLite data and official document files.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `apps/api/internal/db/migrations/000009_billing_documents.sql` | Fiscal series, invoice issue fields, document metadata, client/profile document defaults |
| `apps/api/internal/store/invoice_series.go` | CRUD and sequence formatting for fiscal series |
| `apps/api/internal/store/invoice_documents.go` | Document metadata persistence and invoice issue helpers |
| `apps/api/internal/store/invoice.go` | Extend existing invoice types and draft update behavior |
| `apps/api/internal/billing/snapshot.go` | Build frozen invoice and Work Protocol snapshots |
| `apps/api/internal/billing/issue.go` | Transactional issue orchestration |
| `apps/api/internal/billing/storage.go` | Document root validation, temp writes, final moves, SHA-256 |
| `apps/api/internal/billing/render/renderer.go` | Renderer interface |
| `apps/api/internal/billing/render/html.go` | HTML preview templates |
| `apps/api/internal/billing/render/pdf.go` | PDF renderer implementation |
| `apps/api/internal/httpapi/invoice_series.go` | Fiscal series routes |
| `apps/api/internal/httpapi/invoices.go` | Preview, issue, cancel, documents, download routes |
| `apps/api/internal/httpapi/router.go` | Route registration |
| `apps/api/internal/backup/service.go` | Include documents in backup archive |
| `apps/api/internal/config/config.go` | `LEOTIME_DOCUMENT_ROOT` |
| `apps/web/src/lib/api.ts` | Types and client calls |
| `apps/web/src/lib/invoiceUi.tsx` | Draft, preview, issue, and downloads workflow |
| `apps/web/src/lib/i18n.ts` | Spanish and English UI strings |
| `apps/web/src/styles.css` | Stable document preview and invoice workflow styles |
| `docs/23-invoices-api.md` | Current API update after implementation |
| `docs/31-s3-daily-backups.md` | Backup/restore archive update |
| `docs/06-deploy-vps.md` | Document root and restore expectations |

## Task 1: Schema For Fiscal Series And Documents

**Files:**
- Create: `apps/api/internal/db/migrations/000009_billing_documents.sql`
- Modify: `apps/api/internal/db/migrate_test.go`
- Test: `apps/api/internal/db/migrate_test.go`

**Interfaces:**
- Produces table `invoice_series`.
- Produces table `billing_documents`.
- Extends `invoices` with fiscal fields and document snapshot fields.
- Keeps the existing `invoices.invoice_number` column. Drafts use it as a
  non-official draft reference; issue replaces it with the official number.
- Later tasks consume these exact columns:
  - `invoices.series_id TEXT`
  - `invoices.fiscal_sequence INTEGER`
  - `invoices.period_from TEXT`
  - `invoices.period_to TEXT`
  - `invoices.document_snapshot_json TEXT`
  - `invoices.work_protocol_detail TEXT`
  - `invoice_series.next_sequence INTEGER`
  - `billing_documents.sha256 TEXT`

- [ ] **Step 1: Write the migration**

Create `apps/api/internal/db/migrations/000009_billing_documents.sql`:

```sql
CREATE TABLE invoice_series (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  pattern TEXT NOT NULL DEFAULT '{YYYY}-{SEQ:04}',
  next_sequence INTEGER NOT NULL DEFAULT 1,
  reset_policy TEXT NOT NULL DEFAULT 'yearly',
  active INTEGER NOT NULL DEFAULT 1,
  is_default INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (user_id, code),
  CHECK (next_sequence >= 1),
  CHECK (reset_policy IN ('never', 'yearly')),
  CHECK (active IN (0, 1)),
  CHECK (is_default IN (0, 1))
);

CREATE UNIQUE INDEX idx_invoice_series_one_default
ON invoice_series(user_id)
WHERE is_default = 1;

ALTER TABLE invoices ADD COLUMN series_id TEXT REFERENCES invoice_series(id);
ALTER TABLE invoices ADD COLUMN fiscal_sequence INTEGER;
ALTER TABLE invoices ADD COLUMN period_from TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN period_to TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN document_snapshot_json TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN work_protocol_detail TEXT NOT NULL DEFAULT 'standard'
  CHECK (work_protocol_detail IN ('summary', 'standard', 'detailed'));
ALTER TABLE invoices ADD COLUMN cancelled_at TEXT;
ALTER TABLE invoices ADD COLUMN cancellation_reason TEXT NOT NULL DEFAULT '';

CREATE TABLE billing_documents (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  invoice_id TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  storage_path TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  byte_size INTEGER NOT NULL,
  mime_type TEXT NOT NULL DEFAULT 'application/pdf',
  render_version TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE (invoice_id, kind),
  UNIQUE (user_id, storage_path),
  CHECK (kind IN ('invoice_pdf', 'work_protocol_pdf')),
  CHECK (byte_size > 0),
  CHECK (mime_type = 'application/pdf'),
  CHECK (length(sha256) = 64)
);

ALTER TABLE clients ADD COLUMN default_invoice_series_id TEXT REFERENCES invoice_series(id);
ALTER TABLE clients ADD COLUMN work_protocol_detail TEXT NOT NULL DEFAULT 'standard'
  CHECK (work_protocol_detail IN ('summary', 'standard', 'detailed'));
ALTER TABLE clients ADD COLUMN default_invoice_description TEXT NOT NULL DEFAULT '';

ALTER TABLE app_settings ADD COLUMN seller_tax_id TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN seller_address TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN seller_email TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN payment_instructions TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN default_invoice_series_id TEXT REFERENCES invoice_series(id);
```

- [ ] **Step 2: Run migration tests**

Run:

```bash
cd apps/api && go test ./internal/db -count=1 -v
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/db/migrations/000009_billing_documents.sql apps/api/internal/db/migrate_test.go
git commit -m "feat: add billing document schema"
```

## Task 2: Fiscal Series Store

**Files:**
- Create: `apps/api/internal/store/invoice_series.go`
- Create: `apps/api/internal/store/invoice_series_test.go`

**Interfaces:**
- Produces:
  - `type InvoiceSeries`
  - `type InvoiceSeriesInput`
  - `func (s *Store) ListInvoiceSeries(ctx context.Context, userID string) ([]InvoiceSeries, error)`
  - `func (s *Store) CreateInvoiceSeries(ctx context.Context, userID string, input InvoiceSeriesInput) (*InvoiceSeries, error)`
  - `func (s *Store) UpdateInvoiceSeries(ctx context.Context, userID, seriesID string, input InvoiceSeriesInput) (*InvoiceSeries, error)`
  - `func FormatInvoiceNumber(pattern string, issueTime time.Time, sequence int) (string, error)`
  - `func (s *Store) NextInvoiceNumberTx(ctx context.Context, tx *sql.Tx, userID, seriesID string, issueTime time.Time) (number string, sequence int, err error)`

- [ ] **Step 1: Write failing format tests**

Test cases:

```go
func TestFormatInvoiceNumber(t *testing.T) {
	issueTime := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		pattern  string
		sequence int
		want     string
	}{
		{"year padded", "{YYYY}-{SEQ:04}", 9, "2026-0009"},
		{"invoice prefix", "INV-{YYYY}-{SEQ:03}", 12, "INV-2026-012"},
		{"short year", "{YY}/{SEQ}", 15, "26/15"},
	}
	for _, tc := range cases {
		got, err := FormatInvoiceNumber(tc.pattern, issueTime, tc.sequence)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}
```

Run:

```bash
cd apps/api && go test ./internal/store -run TestFormatInvoiceNumber -count=1 -v
```

Expected: FAIL because `FormatInvoiceNumber` is undefined.

- [ ] **Step 2: Implement formatting and CRUD**

Implement `invoice_series.go` using explicit SQL. Pattern validation accepts
only `{YYYY}`, `{YY}`, `{SEQ}`, and `{SEQ:NN}` where `NN` is 1 to 12.

- [ ] **Step 3: Add transactional sequence test**

Test that two calls to `NextInvoiceNumberTx` inside committed transactions
return `2026-0001` and `2026-0002`, and that a rolled back transaction leaves
the next committed value unchanged.

- [ ] **Step 4: Run store tests**

Run:

```bash
cd apps/api && go test ./internal/store -run 'TestInvoiceSeries|TestFormatInvoiceNumber' -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/store/invoice_series.go apps/api/internal/store/invoice_series_test.go
git commit -m "feat: add configurable invoice series"
```

## Task 3: Document Metadata Store And Invoice Extensions

**Files:**
- Create: `apps/api/internal/store/invoice_documents.go`
- Create: `apps/api/internal/store/invoice_documents_test.go`
- Modify: `apps/api/internal/store/invoice.go`
- Modify: `apps/api/internal/store/invoice_test.go`

**Interfaces:**
- Produces:
  - `type BillingDocument`
  - `type BillingDocumentInput`
  - `func (s *Store) ListInvoiceDocuments(ctx context.Context, userID, invoiceID string) ([]BillingDocument, error)`
  - `func (s *Store) InsertBillingDocumentTx(ctx context.Context, tx *sql.Tx, userID string, input BillingDocumentInput) (*BillingDocument, error)`
  - `func (s *Store) CancelInvoice(ctx context.Context, userID, invoiceID, reason string) (*Invoice, error)`
- Updates `Invoice` JSON shape with `periodFrom`, `periodTo`, `seriesId`, `fiscalSequence`, `workProtocolDetail`, `cancelledAt`, and `documents`.

- [ ] **Step 1: Write document metadata tests**

Tests:

- insert document with valid 64-character hash succeeds,
- duplicate `invoice_id + kind` returns an error,
- invalid hash length returns `ErrInvalidInvoiceInput`,
- path containing `..` returns `ErrInvalidInvoiceInput`.

- [ ] **Step 2: Implement metadata store**

Use lower-case SHA-256 validation:

```go
func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}
```

- [ ] **Step 3: Extend invoice draft inputs**

Add fields to `InvoiceDraftFromTimeInput`:

```go
SeriesID           string `json:"seriesId"`
PeriodFrom         string `json:"periodFrom"`
PeriodTo           string `json:"periodTo"`
WorkProtocolDetail string `json:"workProtocolDetail"`
```

Persist `period_from`, `period_to`, and `work_protocol_detail` when creating
drafts.

- [ ] **Step 4: Run invoice store tests**

Run:

```bash
cd apps/api && go test ./internal/store -run 'TestInvoice|TestBillingDocument' -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/store/invoice.go apps/api/internal/store/invoice_test.go apps/api/internal/store/invoice_documents.go apps/api/internal/store/invoice_documents_test.go
git commit -m "feat: store invoice document metadata"
```

## Task 4: Snapshot Builder

**Files:**
- Create: `apps/api/internal/billing/snapshot.go`
- Create: `apps/api/internal/billing/snapshot_test.go`

**Interfaces:**
- Consumes `store.Invoice` with lines and documents.
- Produces:
  - `type DocumentSnapshot`
  - `type InvoiceSnapshot`
  - `type WorkProtocolSnapshot`
  - `type WorkProtocolDetail string`
  - `func BuildDocumentSnapshot(invoice *store.Invoice, entries []store.TimeEntry, options SnapshotOptions) (DocumentSnapshot, error)`

- [ ] **Step 1: Write snapshot tests for detail levels**

Use synthetic time entries for two days and multiple projects. Assert:

- `summary` rows include date, total quantity, and project names,
- `standard` rows include grouped bullet labels,
- `detailed` rows include entry descriptions and tags when present.

- [ ] **Step 2: Implement snapshot structs**

Use JSON-stable structs:

```go
type DocumentSnapshot struct {
	Version      string               `json:"version"`
	Invoice      InvoiceSnapshot      `json:"invoice"`
	WorkProtocol WorkProtocolSnapshot `json:"workProtocol"`
}
```

Set `Version` to `billing-documents-v1`.

- [ ] **Step 3: Run billing tests**

Run:

```bash
cd apps/api && go test ./internal/billing -run TestBuildDocumentSnapshot -count=1 -v
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/api/internal/billing/snapshot.go apps/api/internal/billing/snapshot_test.go
git commit -m "feat: build billing document snapshots"
```

## Task 5: Document Storage

**Files:**
- Create: `apps/api/internal/billing/storage.go`
- Create: `apps/api/internal/billing/storage_test.go`
- Modify: `apps/api/internal/config/config.go`
- Modify: `apps/api/internal/config/config_test.go`
- Modify: `.env.example`

**Interfaces:**
- Produces:
  - `type DocumentStore`
  - `func NewDocumentStore(root string) (*DocumentStore, error)`
  - `func (s *DocumentStore) WriteOfficial(ctx context.Context, relativePath string, sourcePath string) (StoredDocument, error)`
  - `func (s *DocumentStore) Open(relativePath string) (*os.File, StoredDocument, error)`
  - `type StoredDocument struct { RelativePath string; SHA256 string; ByteSize int64; MIMEType string }`
- Config exposes `DocumentRoot string` from `LEOTIME_DOCUMENT_ROOT`, default `/data/documents`.

- [ ] **Step 1: Write path safety tests**

Cases:

- `invoices/2026/MAIN/2026-0009/invoice.pdf` is accepted,
- `../leotime.db` is rejected,
- `/etc/passwd` is rejected,
- `invoices/2026/x.txt` is rejected because extension is not `.pdf`.

- [ ] **Step 2: Implement storage**

Write to a temp file in the target directory, fsync, compute SHA-256, verify the
first bytes are `%PDF`, and rename into place. Return `application/pdf`.

- [ ] **Step 3: Run storage tests**

Run:

```bash
cd apps/api && go test ./internal/billing -run TestDocumentStore -count=1 -v
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/api/internal/billing/storage.go apps/api/internal/billing/storage_test.go apps/api/internal/config/config.go apps/api/internal/config/config_test.go .env.example
git commit -m "feat: add billing document storage"
```

## Task 6: Preview HTML And PDF Renderer

**Files:**
- Create: `apps/api/internal/billing/render/renderer.go`
- Create: `apps/api/internal/billing/render/html.go`
- Create: `apps/api/internal/billing/render/pdf.go`
- Create: `apps/api/internal/billing/render/render_test.go`

**Interfaces:**
- Produces:
  - `type Renderer interface { RenderPreviewHTML(ctx context.Context, snapshot billing.DocumentSnapshot) ([]byte, error); RenderPDFs(ctx context.Context, snapshot billing.DocumentSnapshot, outputDir string) (RenderedPDFs, error) }`
  - `type RenderedPDFs struct { InvoicePath string; WorkProtocolPath string }`

- [ ] **Step 1: Write preview tests**

Assert preview HTML contains:

- `Invoice #`,
- `Work Protocol #`,
- client name,
- service description,
- total amount,
- table headers `Description`, `Rate Hour`, `Qty`, `Amount`.

- [ ] **Step 2: Implement HTML templates**

Use Go `html/template`. Keep CSS local to the rendered HTML and match the sober
sample style: Letter page, top-right seller block, bordered tables, no
decorative background.

- [ ] **Step 3: Implement PDF renderer behind interface**

Choose the smallest renderer that works inside Docker. The implementation must
write two PDF files and tests must verify they start with `%PDF` and are
non-empty.

- [ ] **Step 4: Run renderer tests**

Run:

```bash
cd apps/api && go test ./internal/billing/render -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/billing/render
git commit -m "feat: render billing previews and PDFs"
```

## Task 7: Transactional Issue Service

**Files:**
- Create: `apps/api/internal/billing/issue.go`
- Create: `apps/api/internal/billing/issue_test.go`
- Modify: `apps/api/internal/store/invoice.go`
- Modify: `apps/api/internal/store/invoice_test.go`

**Interfaces:**
- Consumes fiscal series store, snapshot builder, renderer, document storage.
- Produces:
  - `type IssueService`
  - `type IssueRequest struct { InvoiceID string; IssueAt time.Time }`
  - `func (s *IssueService) Issue(ctx context.Context, userID string, request IssueRequest) (*store.Invoice, error)`

- [ ] **Step 1: Write rollback test**

Use a fake renderer that returns an error. Assert:

- invoice remains `draft`,
- `invoice_number` is still the non-official draft reference,
- fiscal series `next_sequence` is unchanged,
- no `billing_documents` rows exist.

- [ ] **Step 2: Write successful issue test**

Use a fake renderer that writes valid tiny PDF files. Assert:

- invoice status is `issued`,
- invoice number matches selected series,
- `document_snapshot_json` is non-empty valid JSON,
- two `billing_documents` rows exist,
- hashes match file contents,
- update of issued invoice returns `ErrInvoiceNotEditable`.

- [ ] **Step 3: Implement issue orchestration**

Inside one transaction:

```go
number, sequence, err := store.NextInvoiceNumberTx(ctx, tx, userID, invoice.SeriesID, request.IssueAt)
```

Render into a temp directory before final metadata insert. Move files into final
paths only after successful rendering and before commit. If commit fails,
return an error and leave temp files removable by normal temp cleanup.

- [ ] **Step 4: Run issue tests**

Run:

```bash
cd apps/api && go test ./internal/billing -run TestIssue -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/billing/issue.go apps/api/internal/billing/issue_test.go apps/api/internal/store/invoice.go apps/api/internal/store/invoice_test.go
git commit -m "feat: issue immutable billing documents"
```

## Task 8: HTTP API

**Files:**
- Create: `apps/api/internal/httpapi/invoice_series.go`
- Modify: `apps/api/internal/httpapi/invoices.go`
- Modify: `apps/api/internal/httpapi/router.go`
- Modify: `apps/api/internal/httpapi/router_test.go`
- Modify: `apps/api/cmd/leotime/main.go`

**Interfaces:**
- Exposes:
  - `GET /api/v1/invoice-series`
  - `POST /api/v1/invoice-series`
  - `PATCH /api/v1/invoice-series/{seriesID}`
  - `GET /api/v1/invoices/suggestions`
  - `POST /api/v1/invoices/{invoiceID}/preview`
  - `POST /api/v1/invoices/{invoiceID}/issue`
  - `POST /api/v1/invoices/{invoiceID}/cancel`
  - `GET /api/v1/invoices/{invoiceID}/documents`
  - `GET /api/v1/invoices/{invoiceID}/documents/{documentID}/download`

- [ ] **Step 1: Write router tests**

Tests:

- list/create/update series,
- preview returns `text/html`,
- issue returns official number and documents,
- download returns `application/pdf`,
- cancel keeps document download available,
- invalid document ID returns `404`,
- editing issued invoice returns `409`.

- [ ] **Step 2: Wire service dependencies**

Construct `billing.IssueService` in `main.go` using store, renderer, and
document store from config.

- [ ] **Step 3: Implement handlers**

Use existing `writeError` and `writeJSON` patterns. Set download headers:

```text
Content-Type: application/pdf
Content-Disposition: attachment; filename="<invoice-number>-invoice.pdf"
```

- [ ] **Step 4: Run HTTP tests**

Run:

```bash
cd apps/api && go test ./internal/httpapi -run 'TestInvoice|TestInvoiceSeries' -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/httpapi/invoice_series.go apps/api/internal/httpapi/invoices.go apps/api/internal/httpapi/router.go apps/api/internal/httpapi/router_test.go apps/api/cmd/leotime/main.go
git commit -m "feat: expose billing document API"
```

## Task 9: Web API Client And Invoice UI

**Files:**
- Modify: `apps/web/src/lib/api.ts`
- Modify: `apps/web/src/lib/invoiceUi.tsx`
- Modify: `apps/web/src/lib/i18n.ts`
- Modify: `apps/web/src/styles.css`
- Modify: `apps/web/src/App.test.tsx`

**Interfaces:**
- Consumes HTTP API from Task 8.
- Produces UI workflow:
  - draft creation,
  - period suggestions,
  - fiscal series selector,
  - Work Protocol detail selector,
  - preview action,
  - issue action,
  - document downloads,
  - cancellation state with downloads retained.

- [ ] **Step 1: Add TypeScript types**

Add:

```ts
export type WorkProtocolDetail = 'summary' | 'standard' | 'detailed';
export type BillingDocumentKind = 'invoice_pdf' | 'work_protocol_pdf';
export type InvoiceSeries = { id: string; code: string; name: string; pattern: string; nextSequence: number; active: boolean; default: boolean };
export type BillingDocument = { id: string; kind: BillingDocumentKind; sha256: string; byteSize: number; downloadUrl: string };
```

- [ ] **Step 2: Write UI tests**

Use mocked API responses. Assert:

- series selector is shown,
- detail selector offers summary, standard, detailed,
- overlap warning text does not disable issue button,
- issued invoice shows invoice PDF and Work Protocol PDF download buttons.

- [ ] **Step 3: Implement UI flow**

Keep layout dense and work-focused. Avoid a landing page or decorative card
composition. Use buttons with lucide icons for preview, issue, cancel, and
download.

- [ ] **Step 4: Run web tests**

Run:

```bash
npm --workspace @leotime/web run test -- invoice
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/src/lib/api.ts apps/web/src/lib/invoiceUi.tsx apps/web/src/lib/i18n.ts apps/web/src/styles.css apps/web/src/App.test.tsx
git commit -m "feat: add billing document workflow UI"
```

## Task 10: Backup And Restore Documents

**Files:**
- Modify: `apps/api/internal/backup/service.go`
- Modify: `apps/api/internal/backup/service_test.go`
- Modify: `apps/api/internal/backup/snapshot/snapshot.go`
- Modify: `apps/api/internal/backup/snapshot/snapshot_test.go`
- Create: `apps/api/internal/backup/manifest.go`
- Create: `apps/api/internal/backup/manifest_test.go`

**Interfaces:**
- Backup archive contains:
  - `leotime.db`
  - `documents/...`
  - `manifest.json`
- Manifest fields:
  - `generatedAt`
  - `databaseSha256`
  - `documents[] { path, sha256, byteSize }`

- [ ] **Step 1: Write manifest tests**

Assert manifest rejects:

- missing document file,
- wrong hash,
- negative byte size,
- path with `..`.

- [ ] **Step 2: Update backup writer**

Create one gzip tar archive or zip archive containing DB snapshot, documents,
and manifest. Preserve current S3 settings and object listing behavior.

- [ ] **Step 3: Update restore**

Restore into temp DB and temp document root first. Validate manifest and hashes.
Only replace live DB and document root after validation succeeds.

- [ ] **Step 4: Run backup tests**

Run:

```bash
cd apps/api && go test ./internal/backup/... -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/backup
git commit -m "feat: include billing documents in backups"
```

## Task 11: Documentation And Operations

**Files:**
- Modify: `docs/23-invoices-api.md`
- Modify: `docs/31-s3-daily-backups.md`
- Modify: `docs/06-deploy-vps.md`
- Modify: `docs/03-data-model.md`
- Modify: `docs/12-implementation-plan.md`

**Interfaces:**
- Documents current implemented behavior after Tasks 1-10.

- [ ] **Step 1: Update invoices API docs**

Document:

- fiscal series routes,
- preview route,
- issue route,
- cancel route,
- document download routes,
- immutable issued behavior.

- [ ] **Step 2: Update backup docs**

Document archive format, `LEOTIME_DOCUMENT_ROOT`, restore integrity checks, and
operational recovery steps.

- [ ] **Step 3: Update data model docs**

Add `invoice_series`, `billing_documents`, and new invoice fields.

- [ ] **Step 4: Run docs grep check**

Run:

```bash
rg -n "HTML imprimible|guardar como PDF desde el navegador|INV-\\{YYYY\\}" docs
```

Expected: no remaining text that describes the old export as the official PDF
path.

- [ ] **Step 5: Commit**

```bash
git add docs/23-invoices-api.md docs/31-s3-daily-backups.md docs/06-deploy-vps.md docs/03-data-model.md docs/12-implementation-plan.md
git commit -m "docs: document official billing documents"
```

## Task 12: Full Verification

**Files:**
- Modify: `Makefile` if smoke fixtures need a document root
- Modify: smoke test files discovered by `rg -n "smoke|playwright|invoice" .`

**Interfaces:**
- Produces a repeatable local verification path for official invoice PDFs.

- [ ] **Step 1: Run targeted gates**

Run:

```bash
make test-api
make test-web
```

Expected: PASS.

- [ ] **Step 2: Run required full gates**

Run:

```bash
make test
make build-web
make smoke
```

Expected: PASS.

- [ ] **Step 3: Run pre-commit gate**

Run:

```bash
make pre-commit
```

Expected: PASS.

- [ ] **Step 4: Manual smoke checklist**

Using seed data:

```text
1. Create invoice draft from billable time.
2. Preview invoice and Work Protocol.
3. Issue official package.
4. Download invoice PDF.
5. Download Work Protocol PDF.
6. Cancel invoice.
7. Confirm both PDFs still download.
8. Run backup.
9. Restore into fresh local DB and document root.
10. Confirm hashes still match metadata.
```

- [ ] **Step 5: Commit verification updates**

```bash
git add Makefile apps/api apps/web docs
git commit -m "test: verify billing document workflow"
```

## Self-Review

Spec coverage:

- Configurable fiscal series: Tasks 1, 2, 8, 9.
- Official number assigned only on issue: Tasks 2 and 7.
- Preview before export: Tasks 6, 8, 9.
- Persisted official PDFs: Tasks 5, 6, 7.
- Work Protocol with detail levels: Tasks 4, 6, 9.
- Flexible overlapping periods: Tasks 7, 8, 9.
- Immutable issued documents and cancellation: Tasks 3, 7, 8, 9.
- Backup/restore documents: Task 10.
- Documentation updates: Task 11.
- Required gates: Task 12.

Placeholder scan:

- No task uses unspecified placeholder paths.
- No step relies on real client PDFs or production data.
- Follow-up legal features are explicitly out of scope.

Type consistency:

- `WorkProtocolDetail` values are `summary`, `standard`, `detailed` in schema,
  Go, and TypeScript.
- `BillingDocument.kind` values are `invoice_pdf` and `work_protocol_pdf` in
  schema, Go, and TypeScript.
- `LEOTIME_DOCUMENT_ROOT` is the single config name used by storage, backup,
  and docs.
