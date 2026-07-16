ALTER TABLE invoices ADD COLUMN invoice_line_detail TEXT NOT NULL DEFAULT 'granular'
  CHECK (invoice_line_detail IN ('summary', 'by_project', 'granular'));
