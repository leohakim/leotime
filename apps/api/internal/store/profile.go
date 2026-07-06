package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/auth"
)

var ErrProfileNotFound = errors.New("profile not found")
var ErrInvalidProfileInput = errors.New("invalid profile input")
var ErrEmailTaken = errors.New("email already in use")
var ErrInvalidPasswordChange = errors.New("invalid password change")

type AppSettings struct {
	TaskProjectRequired bool   `json:"taskProjectRequired"`
	DefaultCurrency     string `json:"defaultCurrency"`
	Timezone            string `json:"timezone"`
	ThemeMode           string `json:"themeMode"`
}

type Profile struct {
	ID         string      `json:"id"`
	Email      string      `json:"email"`
	Name       string      `json:"name"`
	Locale     string      `json:"locale"`
	LayoutMode string      `json:"layoutMode"`
	Settings   AppSettings `json:"settings"`
	CreatedAt  string      `json:"createdAt"`
	UpdatedAt  string      `json:"updatedAt"`
}

type ProfileUpdateInput struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Locale              string `json:"locale"`
	LayoutMode          string `json:"layoutMode"`
	TaskProjectRequired bool   `json:"taskProjectRequired"`
	DefaultCurrency     string `json:"defaultCurrency"`
	Timezone            string `json:"timezone"`
	ThemeMode           string `json:"themeMode"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Store) ProfileByUserID(ctx context.Context, userID string) (*Profile, error) {
	var profile Profile
	var taskProjectRequired int
	if err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.locale, u.layout_mode, u.created_at, u.updated_at,
			COALESCE(a.task_project_required, 0),
			COALESCE(a.default_currency, 'EUR'),
			COALESCE(a.timezone, 'Europe/Madrid'),
			COALESCE(a.theme_mode, 'solid')
		FROM users u
		LEFT JOIN app_settings a ON a.user_id = u.id
		WHERE u.id = ?
	`, userID).Scan(
		&profile.ID,
		&profile.Email,
		&profile.Name,
		&profile.Locale,
		&profile.LayoutMode,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&taskProjectRequired,
		&profile.Settings.DefaultCurrency,
		&profile.Settings.Timezone,
		&profile.Settings.ThemeMode,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("query profile: %w", err)
	}

	profile.Settings.TaskProjectRequired = taskProjectRequired != 0
	return &profile, nil
}

func (s *Store) UpdateProfile(ctx context.Context, userID string, input ProfileUpdateInput) (*Profile, error) {
	normalized, err := normalizeProfileInput(input)
	if err != nil {
		return nil, err
	}

	var existingEmail string
	if err := s.db.QueryRowContext(ctx, "SELECT email FROM users WHERE id = ?", userID).Scan(&existingEmail); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("query profile user: %w", err)
	}

	if normalized.Email != existingEmail {
		var count int
		if err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM users
			WHERE lower(email) = ? AND id <> ?
		`, normalized.Email, userID).Scan(&count); err != nil {
			return nil, fmt.Errorf("check email uniqueness: %w", err)
		}
		if count > 0 {
			return nil, ErrEmailTaken
		}
	}

	now := nowString()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin profile update: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET email = ?, name = ?, locale = ?, layout_mode = ?, updated_at = ?
		WHERE id = ?
	`, normalized.Email, normalized.Name, normalized.Locale, normalized.LayoutMode, now, userID); err != nil {
		return nil, fmt.Errorf("update profile user: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE app_settings
		SET task_project_required = ?,
			default_currency = ?,
			timezone = ?,
			theme_mode = ?,
			default_locale = ?,
			default_layout_mode = ?,
			updated_at = ?
		WHERE user_id = ?
	`, boolToInt(normalized.TaskProjectRequired), normalized.DefaultCurrency, normalized.Timezone, normalized.ThemeMode, normalized.Locale, normalized.LayoutMode, now, userID)
	if err != nil {
		return nil, fmt.Errorf("update app settings: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("count updated app settings: %w", err)
	}
	if rows == 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO app_settings (
				user_id, task_project_required, default_currency, timezone, theme_mode,
				default_locale, default_layout_mode, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, userID, boolToInt(normalized.TaskProjectRequired), normalized.DefaultCurrency, normalized.Timezone, normalized.ThemeMode, normalized.Locale, normalized.LayoutMode, now); err != nil {
			return nil, fmt.Errorf("insert app settings: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit profile update: %w", err)
	}

	return s.ProfileByUserID(ctx, userID)
}

func (s *Store) ChangePassword(ctx context.Context, userID string, input ChangePasswordInput) error {
	currentPassword := strings.TrimSpace(input.CurrentPassword)
	newPassword := strings.TrimSpace(input.NewPassword)
	if currentPassword == "" || newPassword == "" {
		return fmt.Errorf("%w: current and new password are required", ErrInvalidPasswordChange)
	}
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: new password must be at least 8 characters", ErrInvalidPasswordChange)
	}

	var passwordHash string
	if err := s.db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProfileNotFound
		}
		return fmt.Errorf("query password hash: %w", err)
	}
	if !auth.VerifyPassword(passwordHash, currentPassword) {
		return fmt.Errorf("%w: current password is incorrect", ErrInvalidPasswordChange)
	}

	nextHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = ?, updated_at = ?
		WHERE id = ?
	`, nextHash, nowString(), userID); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}

func normalizeProfileInput(input ProfileUpdateInput) (ProfileUpdateInput, error) {
	normalized := ProfileUpdateInput{
		Name:                strings.TrimSpace(input.Name),
		Email:               strings.TrimSpace(strings.ToLower(input.Email)),
		Locale:              strings.TrimSpace(strings.ToLower(input.Locale)),
		LayoutMode:          strings.TrimSpace(strings.ToLower(input.LayoutMode)),
		TaskProjectRequired: input.TaskProjectRequired,
		DefaultCurrency:     strings.TrimSpace(strings.ToUpper(input.DefaultCurrency)),
		Timezone:            strings.TrimSpace(input.Timezone),
		ThemeMode:           strings.TrimSpace(strings.ToLower(input.ThemeMode)),
	}

	if normalized.Name == "" {
		return ProfileUpdateInput{}, fmt.Errorf("%w: name is required", ErrInvalidProfileInput)
	}
	if normalized.Email == "" {
		return ProfileUpdateInput{}, fmt.Errorf("%w: email is required", ErrInvalidProfileInput)
	}
	if _, err := mail.ParseAddress(normalized.Email); err != nil {
		return ProfileUpdateInput{}, fmt.Errorf("%w: email is invalid", ErrInvalidProfileInput)
	}
	if normalized.Locale != "es" && normalized.Locale != "en" {
		return ProfileUpdateInput{}, fmt.Errorf("%w: locale must be es or en", ErrInvalidProfileInput)
	}
	if normalized.LayoutMode != "solid" && normalized.LayoutMode != "minimal" && normalized.LayoutMode != "compact" {
		return ProfileUpdateInput{}, fmt.Errorf("%w: layoutMode is invalid", ErrInvalidProfileInput)
	}
	if normalized.ThemeMode != "solid" && normalized.ThemeMode != "light" && normalized.ThemeMode != "dark" && normalized.ThemeMode != "minimal" {
		return ProfileUpdateInput{}, fmt.Errorf("%w: themeMode is invalid", ErrInvalidProfileInput)
	}
	if normalized.DefaultCurrency == "" {
		normalized.DefaultCurrency = "EUR"
	}
	if len(normalized.DefaultCurrency) != 3 {
		return ProfileUpdateInput{}, fmt.Errorf("%w: defaultCurrency must be a 3-letter code", ErrInvalidProfileInput)
	}
	if normalized.Timezone == "" {
		normalized.Timezone = "Europe/Madrid"
	}
	if _, err := time.LoadLocation(normalized.Timezone); err != nil {
		return ProfileUpdateInput{}, fmt.Errorf("%w: timezone is invalid", ErrInvalidProfileInput)
	}

	return normalized, nil
}
