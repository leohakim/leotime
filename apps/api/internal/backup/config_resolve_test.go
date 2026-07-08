package backup

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/backup/crypto"
	"github.com/leotime/leotime/apps/api/internal/backup/storage"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestResolveS3ConfigUsesDraftCredentialsWithoutSavedSettings(t *testing.T) {
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

	service := NewService(config.Config{}, st, database, nil)
	service.clientFactory = func(ctx context.Context, cfg storage.S3Config) (storage.Client, error) {
		if cfg.Bucket != "leotime-backups" {
			t.Fatalf("unexpected bucket %q", cfg.Bucket)
		}
		if cfg.AccessKeyID != "leotime_backups" {
			t.Fatalf("unexpected access key %q", cfg.AccessKeyID)
		}
		if cfg.SecretKey != "secret" {
			t.Fatalf("unexpected secret %q", cfg.SecretKey)
		}
		if !storage.EffectivePathStyle("http://minio:9000", cfg.UsePathStyle) {
			t.Fatal("expected path-style for minio endpoint")
		}
		return storage.NewMemoryClient(), nil
	}

	err = service.TestConnection(ctx, user.ID, &store.BackupSettingsInput{
		Enabled:         true,
		Endpoint:        "http://minio:9000",
		Region:          "us-east-1",
		Bucket:          "leotime-backups",
		Prefix:          "leotime/backups/",
		AccessKeyID:     "leotime_backups",
		SecretAccessKey: "secret",
		UsePathStyle:    false,
	})
	if err != nil {
		t.Fatalf("test connection: %v", err)
	}
}

func TestResolveS3ConfigUsesSavedSecretWhenDraftOmitsIt(t *testing.T) {
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
	user, err := st.Authenticate(ctx, "admin@example.com", "change-me-now")
	if err != nil {
		t.Fatal(err)
	}

	secretsKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	service := NewService(config.Config{SecretsKey: secretsKey}, st, database, nil)

	secretEnc, err := crypto.Encrypt([]byte("saved-secret"), []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertBackupSettings(ctx, user.ID, store.BackupSettingsInput{
		Enabled:       true,
		Bucket:        "leotime-backups",
		AccessKeyID:   "leotime_backups",
		Endpoint:      "http://minio:9000",
		ScheduleHour:  1,
		RetentionDays: 365,
	}, secretEnc)
	if err != nil {
		t.Fatal(err)
	}

	service.clientFactory = func(ctx context.Context, cfg storage.S3Config) (storage.Client, error) {
		if cfg.SecretKey != "saved-secret" {
			t.Fatalf("expected saved secret, got %q", cfg.SecretKey)
		}
		return storage.NewMemoryClient(), nil
	}

	err = service.TestConnection(ctx, user.ID, &store.BackupSettingsInput{
		Enabled:     true,
		Endpoint:    "http://minio:9000",
		Bucket:      "leotime-backups",
		AccessKeyID: "leotime_backups",
	})
	if err != nil {
		t.Fatalf("test connection: %v", err)
	}
}
