package store

import (
	"context"
	"fmt"
	"time"
)

type AuthCleanupResult struct {
	Sessions            int64
	PasswordResetTokens int64
}

func (s *Store) PurgeExpiredAuthArtifacts(ctx context.Context, now time.Time) (AuthCleanupResult, error) {
	nowValue := formatTime(now.UTC())

	sessionResult, err := s.db.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE expires_at <= ?
	`, nowValue)
	if err != nil {
		return AuthCleanupResult{}, fmt.Errorf("purge expired sessions: %w", err)
	}
	sessionCount, err := sessionResult.RowsAffected()
	if err != nil {
		return AuthCleanupResult{}, fmt.Errorf("inspect expired session purge: %w", err)
	}

	tokenResult, err := s.db.ExecContext(ctx, `
		DELETE FROM password_reset_tokens
		WHERE expires_at <= ? OR used_at IS NOT NULL
	`, nowValue)
	if err != nil {
		return AuthCleanupResult{}, fmt.Errorf("purge password reset tokens: %w", err)
	}
	tokenCount, err := tokenResult.RowsAffected()
	if err != nil {
		return AuthCleanupResult{}, fmt.Errorf("inspect password reset token purge: %w", err)
	}

	return AuthCleanupResult{
		Sessions:            sessionCount,
		PasswordResetTokens: tokenCount,
	}, nil
}
