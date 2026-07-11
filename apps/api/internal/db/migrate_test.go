package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLatestMigrationVersion(t *testing.T) {
	version, err := LatestMigrationVersion()
	if err != nil {
		t.Fatalf("latest migration version: %v", err)
	}
	if version < 11 {
		t.Fatalf("expected at least migration 000011, got %d", version)
	}
}

func TestMigrateUpgradesVersion2DatabaseWithTagLinks(t *testing.T) {
	ctx := context.Background()
	database := openSyntheticVersion2Database(t, ctx)
	defer database.Close()

	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("migrate from version 2: %v", err)
	}

	latest, err := LatestMigrationVersion()
	if err != nil {
		t.Fatalf("latest migration version: %v", err)
	}
	if err := assertMigrationVersions(ctx, database, latest); err != nil {
		t.Fatal(err)
	}
	if err := assertForeignKeyIntegrity(ctx, database); err != nil {
		t.Fatal(err)
	}
	if err := assertTagsArchiveIndex(ctx, database); err != nil {
		t.Fatal(err)
	}
	if err := assertVersion2TagLinksPreserved(ctx, database); err != nil {
		t.Fatal(err)
	}
}

func openSyntheticVersion2Database(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()

	database, err := Open(ctx, filepath.Join(t.TempDir(), "leotime-v2.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	if _, err := database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		);
	`); err != nil {
		t.Fatalf("create schema migrations table: %v", err)
	}

	for _, fileName := range []string{"000001_init.sql", "000002_import_compat.sql"} {
		if err := execMigrationFile(ctx, database, fileName); err != nil {
			t.Fatalf("apply %s: %v", fileName, err)
		}
	}

	if err := seedVersion2TagFixture(ctx, database); err != nil {
		t.Fatalf("seed version-2 fixture: %v", err)
	}

	for _, row := range []struct {
		version int
		name    string
	}{
		{1, "000001_init.sql"},
		{2, "000002_import_compat.sql"},
	} {
		if _, err := database.ExecContext(ctx, `
			INSERT INTO schema_migrations (version, name) VALUES (?, ?)
		`, row.version, row.name); err != nil {
			t.Fatalf("record migration %d: %v", row.version, err)
		}
	}

	return database
}

func execMigrationFile(ctx context.Context, database *sql.DB, fileName string) error {
	sqlBytes, err := migrationFiles.ReadFile(filepath.Join("migrations", fileName))
	if err != nil {
		return err
	}
	_, err = database.ExecContext(ctx, string(sqlBytes))
	return err
}

func seedVersion2TagFixture(ctx context.Context, database *sql.DB) error {
	const now = "2026-01-01T00:00:00Z"

	_, err := database.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
		VALUES ('user_v2', 'owner@example.com', 'Owner', 'hash', ?, ?)
	`, now, now)
	if err != nil {
		return err
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO tags (id, user_id, name, color, created_at, updated_at) VALUES
		('tag_deep', 'user_v2', 'Deep Work', '#111111', ?, ?),
		('tag_admin', 'user_v2', 'Admin', '#222222', ?, ?)
	`, now, now, now, now)
	if err != nil {
		return err
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO time_entries (
			id, user_id, description, started_at, ended_at, duration_seconds,
			billable, created_at, updated_at
		) VALUES (
			'entry_v2', 'user_v2', 'Tagged work', '2026-01-02T09:00:00Z', '2026-01-02T10:00:00Z',
			3600, 1, ?, ?
		)
	`, now, now)
	if err != nil {
		return err
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO time_entry_tags (time_entry_id, tag_id) VALUES
		('entry_v2', 'tag_deep'),
		('entry_v2', 'tag_admin')
	`)
	return err
}

func assertMigrationVersions(ctx context.Context, database *sql.DB, latest int) error {
	rows, err := database.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return err
	}
	defer rows.Close()

	seen := 0
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return err
		}
		seen++
		if version != seen {
			return errUnexpectedMigrationVersion(seen, version)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if seen != latest {
		return errMigrationCountMismatch(seen, latest)
	}
	return nil
}

func assertForeignKeyIntegrity(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		var rowID int64
		var parent string
		var fkey int
		if err := rows.Scan(&table, &rowID, &parent, &fkey); err != nil {
			return err
		}
		return errForeignKeyViolation(table, rowID)
	}
	return rows.Err()
}

func assertTagsArchiveIndex(ctx context.Context, database *sql.DB) error {
	var indexName string
	err := database.QueryRowContext(ctx, `
		SELECT name
		FROM sqlite_master
		WHERE type = 'index' AND tbl_name = 'tags' AND name = 'idx_tags_user_name_active'
	`).Scan(&indexName)
	if err == sql.ErrNoRows {
		return errMissingTagsArchiveIndex
	}
	if err != nil {
		return err
	}

	var columnName string
	err = database.QueryRowContext(ctx, `
		SELECT name
		FROM pragma_table_info('tags')
		WHERE name = 'archived_at'
	`).Scan(&columnName)
	if err == sql.ErrNoRows {
		return errMissingArchivedAtColumn
	}
	if err != nil {
		return err
	}
	return nil
}

func assertVersion2TagLinksPreserved(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, `
		SELECT t.name
		FROM time_entry_tags tet
		JOIN tags t ON t.id = tet.tag_id
		WHERE tet.time_entry_id = 'entry_v2'
		ORDER BY t.name
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(names) != 2 || names[0] != "Admin" || names[1] != "Deep Work" {
		return errTagLinksChanged(names)
	}

	var archivedAt sql.NullString
	if err := database.QueryRowContext(ctx, `
		SELECT archived_at FROM tags WHERE id = 'tag_deep'
	`).Scan(&archivedAt); err != nil {
		return err
	}
	if archivedAt.Valid {
		return errUnexpectedArchivedTag
	}
	return nil
}

type migrationTestError string

func (e migrationTestError) Error() string { return string(e) }

var (
	errMissingTagsArchiveIndex = migrationTestError("expected idx_tags_user_name_active index on tags")
	errMissingArchivedAtColumn = migrationTestError("expected archived_at column on tags")
	errUnexpectedArchivedTag   = migrationTestError("expected pre-upgrade tags to remain active")
)

func errUnexpectedMigrationVersion(expected, got int) error {
	return migrationTestError("expected migration version " + strconv.Itoa(expected) + ", got " + strconv.Itoa(got))
}

func errMigrationCountMismatch(seen, latest int) error {
	return migrationTestError("expected " + strconv.Itoa(latest) + " applied migrations, got " + strconv.Itoa(seen))
}

func errForeignKeyViolation(table string, rowID int64) error {
	return migrationTestError("foreign key violation in " + table + " row " + strconv.FormatInt(rowID, 10))
}

func errTagLinksChanged(names []string) error {
	return migrationTestError("unexpected tag links after upgrade: " + strings.Join(names, ","))
}
