package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrAISecretsKeyMissing = errors.New("secrets key missing")

type AISettings struct {
	Enabled                 bool    `json:"enabled"`
	GitAuthorEmail          string  `json:"gitAuthorEmail"`
	CursorAPIKeyConfigured  bool    `json:"cursorApiKeyConfigured"`
	CursorCostPerMillionUSD float64 `json:"cursorCostPerMillionUsd"`
}

type AISettingsInput struct {
	Enabled                 bool
	GitAuthorEmail          string
	CursorAPIKey            string
	CursorCostPerMillionUSD float64
}

type AISettingsRecord struct {
	Enabled                 bool
	GitAuthorEmail          string
	CursorAPIKeyEnc         string
	CursorCostPerMillionUSD float64
}

func (s *Store) AISettingsByUserID(ctx context.Context, userID string) (*AISettings, error) {
	record, err := s.aiSettingsRecord(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &AISettings{
		Enabled:                 record.Enabled,
		GitAuthorEmail:          record.GitAuthorEmail,
		CursorAPIKeyConfigured:  strings.TrimSpace(record.CursorAPIKeyEnc) != "",
		CursorCostPerMillionUSD: record.CursorCostPerMillionUSD,
	}, nil
}

func (s *Store) AISettingsRecordByUserID(ctx context.Context, userID string) (*AISettingsRecord, error) {
	return s.aiSettingsRecord(ctx, userID)
}

func (s *Store) UpsertAISettings(ctx context.Context, userID string, input AISettingsInput, cursorKeyEnc string) (*AISettings, error) {
	input.GitAuthorEmail = strings.TrimSpace(input.GitAuthorEmail)
	if input.GitAuthorEmail != "" && !strings.Contains(input.GitAuthorEmail, "@") {
		return nil, validationError(ErrInvalidProfileInput, "gitAuthorEmail", "invalid", "gitAuthorEmail must be an email")
	}

	existing, err := s.aiSettingsRecord(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cursorKeyEnc == "" {
		cursorKeyEnc = existing.CursorAPIKeyEnc
	}

	if input.CursorCostPerMillionUSD <= 0 {
		input.CursorCostPerMillionUSD = existing.CursorCostPerMillionUSD
	}
	if input.CursorCostPerMillionUSD <= 0 {
		input.CursorCostPerMillionUSD = 2
	}

	now := nowString()
	result, err := s.db.ExecContext(ctx, `
		UPDATE app_settings
		SET ai_summary_enabled = ?, git_author_email = ?, cursor_api_key_enc = ?,
			cursor_cost_per_million_usd = ?, updated_at = ?
		WHERE user_id = ?
	`, boolToInt(input.Enabled), input.GitAuthorEmail, cursorKeyEnc, input.CursorCostPerMillionUSD, now, userID)
	if err != nil {
		return nil, fmt.Errorf("update ai settings: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("count ai settings update: %w", err)
	}
	if rows == 0 {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO app_settings (
				user_id, ai_summary_enabled, git_author_email, cursor_api_key_enc,
				cursor_cost_per_million_usd, updated_at
			) VALUES (?, ?, ?, ?, ?, ?)
		`, userID, boolToInt(input.Enabled), input.GitAuthorEmail, cursorKeyEnc, input.CursorCostPerMillionUSD, now); err != nil {
			return nil, fmt.Errorf("insert ai settings: %w", err)
		}
	}
	return s.AISettingsByUserID(ctx, userID)
}

func (s *Store) aiSettingsRecord(ctx context.Context, userID string) (*AISettingsRecord, error) {
	var record AISettingsRecord
	var enabled int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(ai_summary_enabled, 0),
			COALESCE(git_author_email, ''),
			COALESCE(cursor_api_key_enc, ''),
			COALESCE(cursor_cost_per_million_usd, 2.0)
		FROM app_settings
		WHERE user_id = ?
	`, userID).Scan(&enabled, &record.GitAuthorEmail, &record.CursorAPIKeyEnc, &record.CursorCostPerMillionUSD); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &AISettingsRecord{CursorCostPerMillionUSD: 2}, nil
		}
		return nil, fmt.Errorf("query ai settings: %w", err)
	}
	record.Enabled = enabled != 0
	return &record, nil
}
