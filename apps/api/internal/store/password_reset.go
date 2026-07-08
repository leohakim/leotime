package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/auth"
)

var ErrUserNotFound = errors.New("user not found")
var ErrInvalidPasswordReset = errors.New("invalid password reset")

func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, ErrUserNotFound
	}

	var user User
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, locale, layout_mode, created_at, updated_at
		FROM users
		WHERE email = ?
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Locale,
		&user.LayoutMode,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	return &user, nil
}

func (s *Store) CreatePasswordResetToken(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", fmt.Errorf("user id is required")
	}
	if ttl <= 0 {
		ttl = time.Hour
	}

	rawToken, err := randomToken()
	if err != nil {
		return "", err
	}

	tokenID, err := newID("prt")
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	nowString := formatTime(now)

	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM password_reset_tokens
		WHERE user_id = ? AND used_at IS NULL
	`, userID); err != nil {
		return "", fmt.Errorf("clear password reset tokens: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, used_at, created_at)
		VALUES (?, ?, ?, ?, NULL, ?)
	`, tokenID, userID, hashToken(rawToken), formatTime(expiresAt), nowString); err != nil {
		return "", fmt.Errorf("insert password reset token: %w", err)
	}

	return rawToken, nil
}

func (s *Store) ResetPasswordWithToken(ctx context.Context, rawToken string, newPassword string) error {
	rawToken = strings.TrimSpace(rawToken)
	newPassword = strings.TrimSpace(newPassword)
	if rawToken == "" {
		return validationError(ErrInvalidPasswordReset, "token", "required", "token is required")
	}
	if len(newPassword) < 8 {
		return validationError(ErrInvalidPasswordReset, "newPassword", "invalid", "new password must be at least 8 characters")
	}

	var tokenID string
	var userID string
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id
		FROM password_reset_tokens
		WHERE token_hash = ?
			AND used_at IS NULL
			AND expires_at > ?
	`, hashToken(rawToken), nowString()).Scan(&tokenID, &userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidPasswordReset
		}
		return fmt.Errorf("query password reset token: %w", err)
	}

	nextHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin password reset: %w", err)
	}
	defer tx.Rollback()

	now := nowString()
	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET password_hash = ?, updated_at = ?
		WHERE id = ?
	`, nextHash, now, userID); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE password_reset_tokens
		SET used_at = ?
		WHERE id = ?
	`, now, tokenID); err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("clear sessions after password reset: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit password reset: %w", err)
	}
	return nil
}
