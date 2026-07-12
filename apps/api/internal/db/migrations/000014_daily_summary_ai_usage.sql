CREATE TABLE daily_summary_ai_runs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    summary_date TEXT NOT NULL,
    client_id TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL DEFAULT '',
    record_id TEXT,
    model_id TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'cursor',
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_daily_summary_ai_runs_user_created ON daily_summary_ai_runs(user_id, created_at);
CREATE INDEX idx_daily_summary_ai_runs_user_date ON daily_summary_ai_runs(user_id, summary_date);

ALTER TABLE app_settings ADD COLUMN cursor_cost_per_million_usd REAL NOT NULL DEFAULT 2.0;
