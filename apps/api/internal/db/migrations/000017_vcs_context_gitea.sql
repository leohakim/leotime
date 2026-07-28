CREATE TABLE vcs_connections (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('gitea')),
    base_url TEXT NOT NULL,
    owner_identity TEXT NOT NULL DEFAULT '',
    token_enc TEXT NOT NULL DEFAULT '',
    default_client_id TEXT REFERENCES clients(id) ON DELETE SET NULL,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(user_id, provider, base_url)
);

CREATE TABLE vcs_repositories (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id TEXT NOT NULL REFERENCES vcs_connections(id) ON DELETE CASCADE,
    owner TEXT NOT NULL,
    name TEXT NOT NULL,
    client_id TEXT REFERENCES clients(id) ON DELETE SET NULL,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    include_commits INTEGER NOT NULL DEFAULT 1 CHECK (include_commits IN (0, 1)),
    include_pull_requests INTEGER NOT NULL DEFAULT 1 CHECK (include_pull_requests IN (0, 1)),
    include_reviews INTEGER NOT NULL DEFAULT 1 CHECK (include_reviews IN (0, 1)),
    include_issues INTEGER NOT NULL DEFAULT 1 CHECK (include_issues IN (0, 1)),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(connection_id, owner, name)
);

CREATE INDEX idx_vcs_connections_user_enabled ON vcs_connections(user_id, enabled);
CREATE INDEX idx_vcs_repositories_user_connection ON vcs_repositories(user_id, connection_id);
