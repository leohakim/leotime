PRAGMA foreign_keys=OFF;

CREATE TABLE tags_new (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#64748b',
  archived_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

INSERT INTO tags_new (id, user_id, name, color, archived_at, created_at, updated_at)
SELECT id, user_id, name, color, NULL, created_at, updated_at FROM tags;

DROP TABLE tags;

ALTER TABLE tags_new RENAME TO tags;

CREATE UNIQUE INDEX idx_tags_user_name_active ON tags(user_id, name) WHERE archived_at IS NULL;

PRAGMA foreign_keys=ON;
