package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrBackupSettingsNotFound  = errors.New("backup settings not found")
	ErrInvalidBackupSettings   = errors.New("invalid backup settings")
	ErrBackupSecretsKeyMissing = errors.New("backup secrets key missing")
)

const (
	defaultBackupPrefix        = "leotime/backups/"
	defaultBackupScheduleHour  = 1
	defaultBackupRetentionDays = 365
)

type BackupSettings struct {
	Enabled                   bool    `json:"enabled"`
	Endpoint                  string  `json:"endpoint"`
	Region                    string  `json:"region"`
	Bucket                    string  `json:"bucket"`
	Prefix                    string  `json:"prefix"`
	AccessKeyID               string  `json:"accessKeyId"`
	SecretAccessKeyConfigured bool    `json:"secretAccessKeyConfigured"`
	UsePathStyle              bool    `json:"usePathStyle"`
	ScheduleHour              int     `json:"scheduleHour"`
	RetentionDays             int     `json:"retentionDays"`
	LastRunAt                 *string `json:"lastRunAt"`
	LastStatus                string  `json:"lastStatus"`
	LastError                 string  `json:"lastError"`
	LastObjectKey             string  `json:"lastObjectKey"`
	LastRestoreAt             *string `json:"lastRestoreAt"`
	LastRestoreStatus         string  `json:"lastRestoreStatus"`
	LastRestoreError          string  `json:"lastRestoreError"`
	LastRestoreObjectKey      string  `json:"lastRestoreObjectKey"`
}

type BackupSettingsInput struct {
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	UsePathStyle    bool   `json:"usePathStyle"`
	ScheduleHour    int    `json:"scheduleHour"`
	RetentionDays   int    `json:"retentionDays"`
}

type BackupSettingsRecord struct {
	BackupSettings
	SecretAccessKeyEnc string
}

func (s *Store) BackupSettingsByUserID(ctx context.Context, userID string) (*BackupSettings, error) {
	record, err := s.backupSettingsRecordByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &record.BackupSettings, nil
}

func (s *Store) BackupSettingsRecordByUserID(ctx context.Context, userID string) (*BackupSettingsRecord, error) {
	return s.backupSettingsRecordByUserID(ctx, userID)
}

func (s *Store) UpsertBackupSettings(ctx context.Context, userID string, input BackupSettingsInput, secretEnc string) (*BackupSettings, error) {
	existing, existingErr := s.backupSettingsRecordByUserID(ctx, userID)
	if existingErr != nil && !errors.Is(existingErr, ErrBackupSettingsNotFound) {
		return nil, existingErr
	}

	if secretEnc == "" && existing != nil {
		secretEnc = existing.SecretAccessKeyEnc
	}

	hasSecret := strings.TrimSpace(secretEnc) != ""
	normalized, err := normalizeBackupSettingsInput(input, hasSecret)
	if err != nil {
		return nil, err
	}

	if normalized.Enabled && !hasSecret {
		return nil, fmt.Errorf("%w: secret access key is required when enabled", ErrInvalidBackupSettings)
	}

	now := nowString()
	if existing == nil {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO backup_settings (
				user_id, enabled, endpoint, region, bucket, prefix, access_key_id, secret_access_key_enc,
				use_path_style, schedule_hour, retention_days, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, userID, boolToInt(normalized.Enabled), normalized.Endpoint, normalized.Region, normalized.Bucket,
			normalized.Prefix, normalized.AccessKeyID, secretEnc, boolToInt(normalized.UsePathStyle),
			normalized.ScheduleHour, normalized.RetentionDays, now); err != nil {
			return nil, fmt.Errorf("insert backup settings: %w", err)
		}
		return s.BackupSettingsByUserID(ctx, userID)
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE backup_settings
		SET enabled = ?, endpoint = ?, region = ?, bucket = ?, prefix = ?, access_key_id = ?,
			secret_access_key_enc = ?, use_path_style = ?, schedule_hour = ?, retention_days = ?, updated_at = ?
		WHERE user_id = ?
	`, boolToInt(normalized.Enabled), normalized.Endpoint, normalized.Region, normalized.Bucket,
		normalized.Prefix, normalized.AccessKeyID, secretEnc, boolToInt(normalized.UsePathStyle),
		normalized.ScheduleHour, normalized.RetentionDays, now, userID); err != nil {
		return nil, fmt.Errorf("update backup settings: %w", err)
	}

	return s.BackupSettingsByUserID(ctx, userID)
}

func (s *Store) UpdateBackupRunStatus(ctx context.Context, userID, status, errMsg, objectKey string) error {
	now := nowString()
	_, err := s.db.ExecContext(ctx, `
		UPDATE backup_settings
		SET last_run_at = ?, last_status = ?, last_error = ?, last_object_key = ?, updated_at = ?
		WHERE user_id = ?
	`, now, status, errMsg, objectKey, now, userID)
	if err != nil {
		return fmt.Errorf("update backup run status: %w", err)
	}
	return nil
}

func (s *Store) UpdateBackupRestoreStatus(ctx context.Context, userID, status, errMsg, objectKey string) error {
	now := nowString()
	_, err := s.db.ExecContext(ctx, `
		UPDATE backup_settings
		SET last_restore_at = ?, last_restore_status = ?, last_restore_error = ?, last_restore_object_key = ?, updated_at = ?
		WHERE user_id = ?
	`, now, status, errMsg, objectKey, now, userID)
	if err != nil {
		return fmt.Errorf("update backup restore status: %w", err)
	}
	return nil
}

func (s *Store) backupSettingsRecordByUserID(ctx context.Context, userID string) (*BackupSettingsRecord, error) {
	var record BackupSettingsRecord
	var enabled int
	var usePathStyle int
	var lastRunAt sql.NullString
	var lastRestoreAt sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT enabled, endpoint, region, bucket, prefix, access_key_id, secret_access_key_enc,
			use_path_style, schedule_hour, retention_days, last_run_at, last_status, last_error,
			last_object_key, last_restore_at, last_restore_status, last_restore_error, last_restore_object_key
		FROM backup_settings
		WHERE user_id = ?
	`, userID).Scan(
		&enabled,
		&record.Endpoint,
		&record.Region,
		&record.Bucket,
		&record.Prefix,
		&record.AccessKeyID,
		&record.SecretAccessKeyEnc,
		&usePathStyle,
		&record.ScheduleHour,
		&record.RetentionDays,
		&lastRunAt,
		&record.LastStatus,
		&record.LastError,
		&record.LastObjectKey,
		&lastRestoreAt,
		&record.LastRestoreStatus,
		&record.LastRestoreError,
		&record.LastRestoreObjectKey,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBackupSettingsNotFound
		}
		return nil, fmt.Errorf("query backup settings: %w", err)
	}

	record.Enabled = enabled != 0
	record.UsePathStyle = usePathStyle != 0
	record.SecretAccessKeyConfigured = strings.TrimSpace(record.SecretAccessKeyEnc) != ""
	if lastRunAt.Valid {
		value := lastRunAt.String
		record.LastRunAt = &value
	}
	if lastRestoreAt.Valid {
		value := lastRestoreAt.String
		record.LastRestoreAt = &value
	}

	return &record, nil
}

func DefaultBackupSettings() BackupSettings {
	return BackupSettings{
		Prefix:            defaultBackupPrefix,
		ScheduleHour:      defaultBackupScheduleHour,
		RetentionDays:     defaultBackupRetentionDays,
		LastStatus:        "never",
		LastRestoreStatus: "never",
	}
}

func NormalizeBackupSettingsInput(input BackupSettingsInput, hasSecret bool) (BackupSettingsInput, error) {
	return normalizeBackupSettingsInput(input, hasSecret)
}

func normalizeBackupSettingsInput(input BackupSettingsInput, hasSecret bool) (BackupSettingsInput, error) {
	normalized := BackupSettingsInput{
		Enabled:         input.Enabled,
		Endpoint:        strings.TrimSpace(input.Endpoint),
		Region:          strings.TrimSpace(input.Region),
		Bucket:          strings.TrimSpace(input.Bucket),
		Prefix:          strings.TrimSpace(input.Prefix),
		AccessKeyID:     strings.TrimSpace(input.AccessKeyID),
		SecretAccessKey: strings.TrimSpace(input.SecretAccessKey),
		UsePathStyle:    input.UsePathStyle,
		ScheduleHour:    input.ScheduleHour,
		RetentionDays:   input.RetentionDays,
	}

	if normalized.Prefix == "" {
		normalized.Prefix = defaultBackupPrefix
	}
	if !strings.HasSuffix(normalized.Prefix, "/") {
		normalized.Prefix += "/"
	}
	if normalized.ScheduleHour == 0 && !input.Enabled {
		normalized.ScheduleHour = defaultBackupScheduleHour
	} else if normalized.ScheduleHour == 0 {
		normalized.ScheduleHour = defaultBackupScheduleHour
	}
	if normalized.RetentionDays == 0 {
		normalized.RetentionDays = defaultBackupRetentionDays
	}

	if normalized.ScheduleHour < 0 || normalized.ScheduleHour > 23 {
		return BackupSettingsInput{}, fmt.Errorf("%w: scheduleHour must be between 0 and 23", ErrInvalidBackupSettings)
	}
	if normalized.RetentionDays < 1 || normalized.RetentionDays > 3650 {
		return BackupSettingsInput{}, fmt.Errorf("%w: retentionDays must be between 1 and 3650", ErrInvalidBackupSettings)
	}

	if normalized.Enabled {
		if normalized.Bucket == "" || normalized.AccessKeyID == "" {
			return BackupSettingsInput{}, fmt.Errorf("%w: bucket and accessKeyId are required when enabled", ErrInvalidBackupSettings)
		}
		if normalized.SecretAccessKey == "" && !hasSecret {
			return BackupSettingsInput{}, fmt.Errorf("%w: secretAccessKey is required when enabled", ErrInvalidBackupSettings)
		}
	}

	return normalized, nil
}

func (s *Store) EnsureBackupSettingsDefaults(ctx context.Context, userID string) (*BackupSettings, error) {
	settings, err := s.BackupSettingsByUserID(ctx, userID)
	if err == nil {
		return settings, nil
	}
	if !errors.Is(err, ErrBackupSettingsNotFound) {
		return nil, err
	}

	defaults := DefaultBackupSettings()
	now := nowString()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO backup_settings (
			user_id, enabled, endpoint, region, bucket, prefix, access_key_id, secret_access_key_enc,
			use_path_style, schedule_hour, retention_days, last_status, last_restore_status, updated_at
		) VALUES (?, 0, '', '', '', ?, '', '', 0, ?, ?, 'never', 'never', ?)
	`, userID, defaults.Prefix, defaults.ScheduleHour, defaults.RetentionDays, now); err != nil {
		return nil, fmt.Errorf("insert default backup settings: %w", err)
	}

	return s.BackupSettingsByUserID(ctx, userID)
}

func formatBackupTimestamp(now time.Time) string {
	return now.UTC().Format("20060102T150405Z")
}
