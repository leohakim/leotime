ALTER TABLE app_settings ADD COLUMN backup_email_on_success INTEGER NOT NULL DEFAULT 0;
ALTER TABLE app_settings ADD COLUMN backup_email_on_failure INTEGER NOT NULL DEFAULT 1;
ALTER TABLE app_settings ADD COLUMN restore_email_on_success INTEGER NOT NULL DEFAULT 0;
ALTER TABLE app_settings ADD COLUMN restore_email_on_failure INTEGER NOT NULL DEFAULT 1;

CREATE TABLE email_outbox_new (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  time_entry_id TEXT REFERENCES time_entries(id) ON DELETE SET NULL,
  kind TEXT NOT NULL,
  to_address TEXT NOT NULL,
  subject TEXT NOT NULL,
  body_text TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 5,
  next_retry_at TEXT NOT NULL,
  last_error TEXT NOT NULL DEFAULT '',
  sent_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (status IN ('pending', 'sent', 'dead')),
  CHECK (attempts >= 0),
  CHECK (max_attempts >= 1),
  CHECK (kind IN (
    'timer_still_running',
    'password_reset',
    'backup_success',
    'backup_failure',
    'restore_success',
    'restore_failure'
  ))
);

INSERT INTO email_outbox_new SELECT * FROM email_outbox;

DROP TABLE email_outbox;

ALTER TABLE email_outbox_new RENAME TO email_outbox;

CREATE UNIQUE INDEX idx_email_outbox_dedup
  ON email_outbox(kind, time_entry_id)
  WHERE time_entry_id IS NOT NULL AND status IN ('pending', 'sent');

CREATE INDEX idx_email_outbox_pending_retry
  ON email_outbox(status, next_retry_at);
