package db

import "testing"

func TestLatestMigrationVersion(t *testing.T) {
	version, err := LatestMigrationVersion()
	if err != nil {
		t.Fatalf("latest migration version: %v", err)
	}
	if version < 11 {
		t.Fatalf("expected at least migration 000011, got %d", version)
	}
}
