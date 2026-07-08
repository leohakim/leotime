package backup

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/backup/crypto"
	"github.com/leotime/leotime/apps/api/internal/backup/storage"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestServiceRunAndRestore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "leotime.db")

	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	st := store.New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	user, err := st.Authenticate(ctx, "admin@example.com", "change-me-now")
	if err != nil {
		t.Fatalf("auth: %v", err)
	}

	secretsKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	cfg := config.Config{
		DBPath:         dbPath,
		SecretsKey:     secretsKey,
		BootstrapEmail: "admin@example.com",
	}

	memory := storage.NewMemoryClient()
	service := NewService(cfg, st, database, nil)
	service.clientFactory = func(ctx context.Context, cfg storage.S3Config) (storage.Client, error) {
		return memory, nil
	}

	secretEnc, err := crypto.Encrypt([]byte("secret"), []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err = st.UpsertBackupSettings(ctx, user.ID, store.BackupSettingsInput{
		Enabled:       true,
		Bucket:        "bucket",
		AccessKeyID:   "key",
		ScheduleHour:  1,
		RetentionDays: 365,
	}, secretEnc)
	if err != nil {
		t.Fatalf("save settings: %v", err)
	}

	if _, err := database.ExecContext(ctx, `
		INSERT INTO clients (id, user_id, name, default_currency, default_hourly_rate_minor, created_at, updated_at)
		VALUES ('cli_test', ?, 'ACME', 'EUR', 7500, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`, user.ID); err != nil {
		t.Fatalf("insert client: %v", err)
	}

	runResult, err := service.Run(ctx, user.ID, true)
	if err != nil {
		t.Fatalf("run backup: %v", err)
	}
	if runResult.Status != "success" || runResult.ObjectKey == "" {
		t.Fatalf("unexpected run result: %+v", runResult)
	}
	if len(memory.Objects) != 1 {
		t.Fatalf("expected one uploaded object, got %d", len(memory.Objects))
	}

	if _, err := database.ExecContext(ctx, "DELETE FROM clients WHERE id = 'cli_test'"); err != nil {
		t.Fatalf("delete client: %v", err)
	}

	restoreResult, err := service.Restore(ctx, user.ID, runResult.ObjectKey, false)
	if err != nil {
		t.Fatalf("restore backup: %v", err)
	}
	if restoreResult.Status != "success" {
		t.Fatalf("unexpected restore result: %+v", restoreResult)
	}

	var count int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM clients WHERE id = 'cli_test'").Scan(&count); err != nil {
		t.Fatalf("count clients: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected restored client, got count %d", count)
	}
}

func TestServiceRunScheduledSkipsWhenDisabled(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "leotime.db")
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	st := store.New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		DBPath:                 dbPath,
		BootstrapEmail:         "admin@example.com",
		BackupSchedulerEnabled: false,
	}
	service := NewService(cfg, st, database, nil)
	if err := service.RunScheduled(ctx); err != nil {
		t.Fatalf("scheduled run: %v", err)
	}
}

func TestIsDueUsesTimezone(t *testing.T) {
	lastRun := time.Date(2026, 7, 6, 2, 0, 0, 0, time.FixedZone("Madrid", 2*3600)).Format(time.RFC3339)
	settings := EnabledSettings{
		Enabled:      true,
		ScheduleHour: 1,
		LastRunAt:    &lastRun,
		LastStatus:   "success",
	}
	now := time.Date(2026, 7, 6, 22, 30, 0, 0, time.UTC)
	due, err := IsDue(settings, "Europe/Madrid", now, false)
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("expected not due before 01:00 Madrid time")
	}
}
