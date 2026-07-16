ALTER TABLE app_settings ADD COLUMN invoice_withholding_label TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN withholding_label TEXT NOT NULL DEFAULT '';
