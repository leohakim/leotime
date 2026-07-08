package snapshot

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func SnapshotToFile(ctx context.Context, dbPath, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf("VACUUM INTO %q", destPath)
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("vacuum into snapshot: %w", err)
	}

	return nil
}

func GzipFile(srcPath, destPath string) error {
	source, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create gzip directory: %w", err)
	}

	destination, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create gzip destination: %w", err)
	}
	defer destination.Close()

	writer := gzip.NewWriter(destination)
	if _, err := io.Copy(writer, source); err != nil {
		writer.Close()
		return fmt.Errorf("gzip copy: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}
	if err := destination.Close(); err != nil {
		return fmt.Errorf("close gzip destination: %w", err)
	}

	return nil
}

func GunzipToFile(srcPath, destPath string) error {
	source, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open gzip source: %w", err)
	}
	defer source.Close()

	reader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("new gzip reader: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create gunzip directory: %w", err)
	}

	destination, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create gunzip destination: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, reader); err != nil {
		return fmt.Errorf("gunzip copy: %w", err)
	}

	return nil
}

func ValidateDatabase(ctx context.Context, dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open validation database: %w", err)
	}
	defer db.Close()

	requiredTables := []string{"users", "clients", "time_entries"}
	for _, table := range requiredTables {
		var name string
		if err := db.QueryRowContext(ctx, `
			SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?
		`, table).Scan(&name); err != nil {
			return fmt.Errorf("missing table %q: %w", table, err)
		}
	}

	return nil
}
