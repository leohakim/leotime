PRAGMA foreign_keys=OFF;

CREATE TABLE daily_summary_records_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    summary_date TEXT NOT NULL,
    client_id TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    draft_text TEXT NOT NULL DEFAULT '',
    approved_text TEXT NOT NULL DEFAULT '',
    manual_note TEXT NOT NULL DEFAULT '',
    options_json TEXT NOT NULL DEFAULT '{}',
    generation_source TEXT NOT NULL DEFAULT 'template',
    generation_count INTEGER NOT NULL DEFAULT 0,
    context_json TEXT NOT NULL DEFAULT '{}',
    approved_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(user_id, summary_date, client_id, project_id)
);

INSERT INTO daily_summary_records_new (
    id, user_id, summary_date, client_id, project_id, status, draft_text, approved_text,
    manual_note, options_json, generation_source, generation_count, context_json,
    approved_at, created_at, updated_at
)
SELECT
    id, user_id, summary_date, '', '', status, draft_text, approved_text,
    manual_note, options_json, generation_source, generation_count, context_json,
    approved_at, created_at, updated_at
FROM daily_summary_records;

DROP TABLE daily_summary_records;

ALTER TABLE daily_summary_records_new RENAME TO daily_summary_records;

CREATE INDEX idx_daily_summary_records_user_scope ON daily_summary_records(user_id, summary_date, client_id, project_id);

PRAGMA foreign_keys=ON;
