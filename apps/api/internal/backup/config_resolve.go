package backup

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/backup/storage"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type resolvedS3Config struct {
	record *store.BackupSettingsRecord
	cfg    storage.S3Config
	prefix string
}

func (s *Service) TestConnection(ctx context.Context, userID string, draft *store.BackupSettingsInput) error {
	resolved, err := s.resolveS3Config(ctx, userID, draft, false)
	if err != nil {
		return err
	}

	client, err := s.clientFactory(ctx, resolved.cfg)
	if err != nil {
		return fmt.Errorf("create s3 client: %w", err)
	}

	testKey := resolved.prefix + "leotime-connection-test.txt"
	if err := client.Put(ctx, testKey, strings.NewReader("leotime backup connection test"), "text/plain"); err != nil {
		return err
	}
	if err := client.Delete(ctx, testKey); err != nil {
		return err
	}
	return nil
}

func (s *Service) resolveS3Config(ctx context.Context, userID string, draft *store.BackupSettingsInput, requireEnabled bool) (*resolvedS3Config, error) {
	record, err := s.store.BackupSettingsRecordByUserID(ctx, userID)
	if err != nil && !errors.Is(err, store.ErrBackupSettingsNotFound) {
		return nil, err
	}

	input := store.BackupSettingsInput{
		Prefix:        "leotime/backups/",
		ScheduleHour:  1,
		RetentionDays: 365,
	}
	if record != nil {
		input = store.BackupSettingsInput{
			Enabled:       record.Enabled,
			Endpoint:      record.Endpoint,
			Region:        record.Region,
			Bucket:        record.Bucket,
			Prefix:        record.Prefix,
			AccessKeyID:   record.AccessKeyID,
			UsePathStyle:  record.UsePathStyle,
			ScheduleHour:  record.ScheduleHour,
			RetentionDays: record.RetentionDays,
		}
	}
	if draft != nil {
		input = mergeBackupSettingsInput(input, *draft)
	}

	normalized, err := store.NormalizeBackupSettingsInput(input, hasConfiguredSecret(record, draft))
	if err != nil {
		return nil, err
	}

	if requireEnabled && !normalized.Enabled {
		return nil, store.BackupSettingsValidationError("enabled", "invalid", "backups are disabled")
	}

	secret, err := s.resolveSecretKey(record, draft)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(normalized.Bucket) == "" || strings.TrimSpace(normalized.AccessKeyID) == "" {
		return nil, store.BackupSettingsValidationError("bucket", "required", "bucket and accessKeyId are required")
	}
	if strings.TrimSpace(secret) == "" {
		return nil, store.BackupSettingsValidationError("secretAccessKey", "required", "secret access key is required")
	}

	return &resolvedS3Config{
		record: record,
		prefix: normalized.Prefix,
		cfg: storage.S3Config{
			Endpoint:     normalized.Endpoint,
			Region:       normalized.Region,
			Bucket:       normalized.Bucket,
			AccessKeyID:  normalized.AccessKeyID,
			SecretKey:    secret,
			UsePathStyle: normalized.UsePathStyle,
		},
	}, nil
}

func mergeBackupSettingsInput(base, draft store.BackupSettingsInput) store.BackupSettingsInput {
	out := draft
	if strings.TrimSpace(out.Prefix) == "" {
		out.Prefix = base.Prefix
	}
	if out.ScheduleHour == 0 {
		out.ScheduleHour = base.ScheduleHour
	}
	if out.RetentionDays == 0 {
		out.RetentionDays = base.RetentionDays
	}
	if strings.TrimSpace(out.SecretAccessKey) == "" {
		out.SecretAccessKey = base.SecretAccessKey
	}
	return out
}

func hasConfiguredSecret(record *store.BackupSettingsRecord, draft *store.BackupSettingsInput) bool {
	if draft != nil && strings.TrimSpace(draft.SecretAccessKey) != "" {
		return true
	}
	return record != nil && strings.TrimSpace(record.SecretAccessKeyEnc) != ""
}

func (s *Service) resolveSecretKey(record *store.BackupSettingsRecord, draft *store.BackupSettingsInput) (string, error) {
	if draft != nil && strings.TrimSpace(draft.SecretAccessKey) != "" {
		return strings.TrimSpace(draft.SecretAccessKey), nil
	}
	if record == nil || strings.TrimSpace(record.SecretAccessKeyEnc) == "" {
		return "", store.BackupSettingsValidationError("secretAccessKey", "required", "secret access key is required")
	}
	return s.decryptSecret(record.SecretAccessKeyEnc)
}
