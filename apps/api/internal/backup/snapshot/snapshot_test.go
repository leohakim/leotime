package snapshot

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/db"
)

func TestSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "leotime.db")

	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, name, locale, layout_mode, created_at, updated_at)
		VALUES ('usr_test', 'test@example.com', 'hash', 'Test', 'es', 'solid', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	database.Close()

	snapshotPath := filepath.Join(dir, "snapshot.db")
	if err := SnapshotToFile(ctx, dbPath, snapshotPath); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	gzipPath := filepath.Join(dir, "snapshot.db.gz")
	if err := GzipFile(snapshotPath, gzipPath); err != nil {
		t.Fatalf("gzip: %v", err)
	}

	restoredPath := filepath.Join(dir, "restored.db")
	if err := GunzipToFile(gzipPath, restoredPath); err != nil {
		t.Fatalf("gunzip: %v", err)
	}
	if err := ValidateDatabase(ctx, restoredPath); err != nil {
		t.Fatalf("validate restored db: %v", err)
	}

	copyDB, err := sql.Open("sqlite", restoredPath)
	if err != nil {
		t.Fatalf("open restored db: %v", err)
	}
	defer copyDB.Close()

	var count int
	if err := copyDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "test@example.com").Scan(&count); err != nil {
		t.Fatalf("query restored db: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one user, got %d", count)
	}

	if _, err := os.Stat(gzipPath); err != nil {
		t.Fatalf("gzip file missing: %v", err)
	}
}
