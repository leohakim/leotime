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
