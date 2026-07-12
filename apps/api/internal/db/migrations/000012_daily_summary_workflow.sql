ALTER TABLE projects ADD COLUMN local_repo_path TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN git_remote_url TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN cursor_workspace_slug TEXT NOT NULL DEFAULT '';

ALTER TABLE app_settings ADD COLUMN cursor_api_key_enc TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN git_author_email TEXT NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN ai_summary_enabled INTEGER NOT NULL DEFAULT 0;

CREATE TABLE daily_summary_records (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    summary_date TEXT NOT NULL,
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
    UNIQUE(user_id, summary_date)
);

CREATE INDEX idx_daily_summary_records_user_date ON daily_summary_records(user_id, summary_date);
