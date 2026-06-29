CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  locale TEXT NOT NULL DEFAULT 'es',
  layout_mode TEXT NOT NULL DEFAULT 'solid',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (locale IN ('es', 'en')),
  CHECK (layout_mode IN ('solid', 'minimal', 'compact'))
);

CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE clients (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  email TEXT,
  tax_id TEXT,
  billing_address TEXT,
  default_currency TEXT NOT NULL DEFAULT 'EUR',
  default_hourly_rate_minor INTEGER NOT NULL DEFAULT 0,
  archived_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_id TEXT REFERENCES clients(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#2563eb',
  default_hourly_rate_minor INTEGER,
  archived_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  billable INTEGER NOT NULL DEFAULT 1,
  archived_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE tags (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#64748b',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (user_id, name)
);

CREATE TABLE time_entries (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_id TEXT REFERENCES clients(id) ON DELETE SET NULL,
  project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
  task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL,
  description TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  ended_at TEXT,
  duration_seconds INTEGER NOT NULL DEFAULT 0,
  billable INTEGER NOT NULL DEFAULT 1,
  overlap_warning INTEGER NOT NULL DEFAULT 0,
  source TEXT NOT NULL DEFAULT 'manual',
  sync_state TEXT NOT NULL DEFAULT 'synced',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (duration_seconds >= 0),
  CHECK (source IN ('manual', 'timer', 'import', 'offline')),
  CHECK (sync_state IN ('synced', 'pending', 'conflict'))
);

CREATE TABLE time_entry_tags (
  time_entry_id TEXT NOT NULL REFERENCES time_entries(id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (time_entry_id, tag_id)
);

CREATE TABLE rates (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_id TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
  currency TEXT NOT NULL DEFAULT 'EUR',
  hourly_rate_minor INTEGER NOT NULL,
  valid_from TEXT,
  valid_to TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (hourly_rate_minor >= 0)
);

CREATE TABLE invoices (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_id TEXT REFERENCES clients(id) ON DELETE SET NULL,
  invoice_number TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  currency TEXT NOT NULL DEFAULT 'EUR',
  issued_at TEXT,
  due_at TEXT,
  seller_name TEXT NOT NULL DEFAULT '',
  seller_tax_id TEXT NOT NULL DEFAULT '',
  seller_address TEXT NOT NULL DEFAULT '',
  client_name TEXT NOT NULL DEFAULT '',
  client_tax_id TEXT NOT NULL DEFAULT '',
  client_address TEXT NOT NULL DEFAULT '',
  subtotal_minor INTEGER NOT NULL DEFAULT 0,
  tax_minor INTEGER NOT NULL DEFAULT 0,
  withholding_minor INTEGER NOT NULL DEFAULT 0,
  total_minor INTEGER NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (user_id, invoice_number),
  CHECK (status IN ('draft', 'issued', 'paid', 'cancelled'))
);

CREATE TABLE invoice_lines (
  id TEXT PRIMARY KEY,
  invoice_id TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
  time_entry_id TEXT REFERENCES time_entries(id) ON DELETE SET NULL,
  description TEXT NOT NULL,
  quantity_minutes INTEGER NOT NULL,
  unit_rate_minor INTEGER NOT NULL,
  subtotal_minor INTEGER NOT NULL,
  tax_rate_basis_points INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  CHECK (quantity_minutes >= 0),
  CHECK (unit_rate_minor >= 0),
  CHECK (subtotal_minor >= 0)
);

CREATE TABLE app_settings (
  user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  task_project_required INTEGER NOT NULL DEFAULT 0,
  default_rounding_minutes INTEGER NOT NULL DEFAULT 1,
  default_locale TEXT NOT NULL DEFAULT 'es',
  default_layout_mode TEXT NOT NULL DEFAULT 'solid',
  updated_at TEXT NOT NULL,
  CHECK (default_rounding_minutes = 1)
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_time_entries_user_started ON time_entries(user_id, started_at);
CREATE INDEX idx_time_entries_project ON time_entries(project_id);
CREATE INDEX idx_time_entries_task ON time_entries(task_id);
CREATE INDEX idx_invoices_user_status ON invoices(user_id, status);

