package store

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/backup/crypto"
)

func TestBackupSettingsDefaultsAndSecretRetention(t *testing.T) {
	ctx := context.Background()
	st, user := newProfileTestStore(t, ctx)

	settings, err := st.EnsureBackupSettingsDefaults(ctx, user.ID)
	if err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	if settings.ScheduleHour != 1 || settings.RetentionDays != 365 {
		t.Fatalf("unexpected defaults: %+v", settings)
	}
	if settings.Prefix != "leotime/backups/" {
		t.Fatalf("unexpected prefix: %q", settings.Prefix)
	}

	key := make([]byte, 32)
	copy(key, []byte("01234567890123456789012345678901"))
	secretEnc, err := crypto.Encrypt([]byte("super-secret"), key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	saved, err := st.UpsertBackupSettings(ctx, user.ID, BackupSettingsInput{
		Enabled:         true,
		Bucket:          "my-bucket",
		AccessKeyID:     "AKIATEST",
		SecretAccessKey: "super-secret",
		ScheduleHour:    1,
		RetentionDays:   365,
	}, secretEnc)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !saved.SecretAccessKeyConfigured {
		t.Fatal("expected configured secret")
	}

	updated, err := st.UpsertBackupSettings(ctx, user.ID, BackupSettingsInput{
		Enabled:       true,
		Bucket:        "my-bucket",
		AccessKeyID:   "AKIAUPDATED",
		ScheduleHour:  2,
		RetentionDays: 180,
	}, "")
	if err != nil {
		t.Fatalf("update without secret: %v", err)
	}
	if updated.AccessKeyID != "AKIAUPDATED" {
		t.Fatalf("expected updated access key, got %q", updated.AccessKeyID)
	}

	record, err := st.BackupSettingsRecordByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load record: %v", err)
	}
	if record.SecretAccessKeyEnc != secretEnc {
		t.Fatal("expected previous encrypted secret to be retained")
	}
}

func TestBackupSettingsValidation(t *testing.T) {
	_, err := normalizeBackupSettingsInput(BackupSettingsInput{
		Enabled:     true,
		Bucket:      "",
		AccessKeyID: "",
	}, false)
	if !errors.Is(err, ErrInvalidBackupSettings) {
		t.Fatalf("expected invalid settings error, got %v", err)
	}

	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	_ = key
}
