CREATE TABLE import_runs (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  source_path TEXT NOT NULL,
  dry_run INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  summary_json TEXT NOT NULL DEFAULT '{}',
  started_at TEXT NOT NULL,
  finished_at TEXT,
  error TEXT,
  CHECK (provider IN ('solidtime')),
  CHECK (status IN ('running', 'completed', 'failed'))
);

CREATE TABLE external_mappings (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  external_type TEXT NOT NULL,
  external_id TEXT NOT NULL,
  internal_type TEXT NOT NULL,
  internal_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (provider, external_type, external_id)
);

CREATE INDEX idx_external_mappings_internal ON external_mappings(provider, internal_type, internal_id);
CREATE INDEX idx_import_runs_provider_started ON import_runs(provider, started_at);

