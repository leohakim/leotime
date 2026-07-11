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

CREATE TABLE time_entry_tags_backup (
  time_entry_id TEXT NOT NULL,
  tag_id TEXT NOT NULL,
  PRIMARY KEY (time_entry_id, tag_id)
);

INSERT INTO time_entry_tags_backup (time_entry_id, tag_id)
SELECT time_entry_id, tag_id FROM time_entry_tags;

DELETE FROM time_entry_tags;

DROP TABLE tags;

ALTER TABLE tags_new RENAME TO tags;

INSERT INTO time_entry_tags (time_entry_id, tag_id)
SELECT time_entry_id, tag_id FROM time_entry_tags_backup;

DROP TABLE time_entry_tags_backup;

CREATE UNIQUE INDEX idx_tags_user_name_active ON tags(user_id, name) WHERE archived_at IS NULL;
