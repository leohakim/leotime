package db

import (
	"context"
	"testing"
)

func TestMigrateCreatesCoreTables(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	var tableCount int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = 'table'
			AND name IN ('users', 'clients', 'projects', 'tasks', 'time_entries', 'invoices');
	`).Scan(&tableCount); err != nil {
		t.Fatalf("count tables: %v", err)
	}

	if tableCount != 6 {
		t.Fatalf("expected 6 core tables, got %d", tableCount)
	}
}
