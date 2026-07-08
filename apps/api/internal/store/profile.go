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
	TaskProjectRequired      bool   `json:"taskProjectRequired"`
	DefaultCurrency          string `json:"defaultCurrency"`
	Timezone                 string `json:"timezone"`
	ThemeMode                string `json:"themeMode"`
	TimerStillRunningEnabled bool   `json:"timerStillRunningEnabled"`
	TimerStillRunningHours   int    `json:"timerStillRunningHours"`
	BackupEmailOnSuccess     bool   `json:"backupEmailOnSuccess"`
	BackupEmailOnFailure     bool   `json:"backupEmailOnFailure"`
	RestoreEmailOnSuccess    bool   `json:"restoreEmailOnSuccess"`
	RestoreEmailOnFailure    bool   `json:"restoreEmailOnFailure"`
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
	Name                     string `json:"name"`
	Email                    string `json:"email"`
	Locale                   string `json:"locale"`
	LayoutMode               string `json:"layoutMode"`
	TaskProjectRequired      bool   `json:"taskProjectRequired"`
	DefaultCurrency          string `json:"defaultCurrency"`
	Timezone                 string `json:"timezone"`
	ThemeMode                string `json:"themeMode"`
	TimerStillRunningEnabled bool   `json:"timerStillRunningEnabled"`
	TimerStillRunningHours   int    `json:"timerStillRunningHours"`
	BackupEmailOnSuccess     bool   `json:"backupEmailOnSuccess"`
	BackupEmailOnFailure     bool   `json:"backupEmailOnFailure"`
	RestoreEmailOnSuccess    bool   `json:"restoreEmailOnSuccess"`
	RestoreEmailOnFailure    bool   `json:"restoreEmailOnFailure"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Store) ProfileByUserID(ctx context.Context, userID string) (*Profile, error) {
	var profile Profile
	var taskProjectRequired int
	var timerStillRunningEnabled int
	var backupEmailOnSuccess int
	var backupEmailOnFailure int
	var restoreEmailOnSuccess int
	var restoreEmailOnFailure int
	if err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.locale, u.layout_mode, u.created_at, u.updated_at,
			COALESCE(a.task_project_required, 0),
			COALESCE(a.default_currency, 'EUR'),
			COALESCE(NULLIF(a.timezone, ''), 'Europe/Madrid'),
			COALESCE(a.theme_mode, 'solid'),
			COALESCE(a.timer_still_running_enabled, 1),
			COALESCE(a.timer_still_running_hours, 8),
			COALESCE(a.backup_email_on_success, 0),
			COALESCE(a.backup_email_on_failure, 1),
			COALESCE(a.restore_email_on_success, 0),
			COALESCE(a.restore_email_on_failure, 1)
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
		&timerStillRunningEnabled,
		&profile.Settings.TimerStillRunningHours,
		&backupEmailOnSuccess,
		&backupEmailOnFailure,
		&restoreEmailOnSuccess,
		&restoreEmailOnFailure,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("query profile: %w", err)
	}

	profile.Settings.TaskProjectRequired = taskProjectRequired != 0
	profile.Settings.TimerStillRunningEnabled = timerStillRunningEnabled != 0
	profile.Settings.BackupEmailOnSuccess = backupEmailOnSuccess != 0
	profile.Settings.BackupEmailOnFailure = backupEmailOnFailure != 0
	profile.Settings.RestoreEmailOnSuccess = restoreEmailOnSuccess != 0
	profile.Settings.RestoreEmailOnFailure = restoreEmailOnFailure != 0
	if profile.Settings.TimerStillRunningHours <= 0 {
		profile.Settings.TimerStillRunningHours = 8
	}
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
			timer_still_running_enabled = ?,
			timer_still_running_hours = ?,
			backup_email_on_success = ?,
			backup_email_on_failure = ?,
			restore_email_on_success = ?,
			restore_email_on_failure = ?,
			default_locale = ?,
			default_layout_mode = ?,
			updated_at = ?
		WHERE user_id = ?
	`, boolToInt(normalized.TaskProjectRequired), normalized.DefaultCurrency, normalized.Timezone, normalized.ThemeMode,
		boolToInt(normalized.TimerStillRunningEnabled), normalized.TimerStillRunningHours,
		boolToInt(normalized.BackupEmailOnSuccess), boolToInt(normalized.BackupEmailOnFailure),
		boolToInt(normalized.RestoreEmailOnSuccess), boolToInt(normalized.RestoreEmailOnFailure),
		normalized.Locale, normalized.LayoutMode, now, userID)
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
				timer_still_running_enabled, timer_still_running_hours,
				backup_email_on_success, backup_email_on_failure,
				restore_email_on_success, restore_email_on_failure,
				default_locale, default_layout_mode, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, userID, boolToInt(normalized.TaskProjectRequired), normalized.DefaultCurrency, normalized.Timezone, normalized.ThemeMode,
			boolToInt(normalized.TimerStillRunningEnabled), normalized.TimerStillRunningHours,
			boolToInt(normalized.BackupEmailOnSuccess), boolToInt(normalized.BackupEmailOnFailure),
			boolToInt(normalized.RestoreEmailOnSuccess), boolToInt(normalized.RestoreEmailOnFailure),
			normalized.Locale, normalized.LayoutMode, now); err != nil {
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
		return validationError(ErrInvalidPasswordChange, "currentPassword", "required", "current and new password are required")
	}
	if len(newPassword) < 8 {
		return validationError(ErrInvalidPasswordChange, "newPassword", "invalid", "new password must be at least 8 characters")
	}

	var passwordHash string
	if err := s.db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProfileNotFound
		}
		return fmt.Errorf("query password hash: %w", err)
	}
	if !auth.VerifyPassword(passwordHash, currentPassword) {
		return validationError(ErrInvalidPasswordChange, "currentPassword", "invalid", "current password is incorrect")
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
		Name:                     strings.TrimSpace(input.Name),
		Email:                    strings.TrimSpace(strings.ToLower(input.Email)),
		Locale:                   strings.TrimSpace(strings.ToLower(input.Locale)),
		LayoutMode:               strings.TrimSpace(strings.ToLower(input.LayoutMode)),
		TaskProjectRequired:      input.TaskProjectRequired,
		DefaultCurrency:          strings.TrimSpace(strings.ToUpper(input.DefaultCurrency)),
		Timezone:                 strings.TrimSpace(input.Timezone),
		ThemeMode:                strings.TrimSpace(strings.ToLower(input.ThemeMode)),
		TimerStillRunningEnabled: input.TimerStillRunningEnabled,
		TimerStillRunningHours:   input.TimerStillRunningHours,
		BackupEmailOnSuccess:     input.BackupEmailOnSuccess,
		BackupEmailOnFailure:     input.BackupEmailOnFailure,
		RestoreEmailOnSuccess:    input.RestoreEmailOnSuccess,
		RestoreEmailOnFailure:    input.RestoreEmailOnFailure,
	}

	if normalized.Name == "" {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "name", "required", "name is required")
	}
	if normalized.Email == "" {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "email", "required", "email is required")
	}
	if _, err := mail.ParseAddress(normalized.Email); err != nil {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "email", "invalid", "email is invalid")
	}
	if normalized.Locale != "es" && normalized.Locale != "en" {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "locale", "invalid", "locale must be es or en")
	}
	if normalized.LayoutMode != "solid" && normalized.LayoutMode != "minimal" && normalized.LayoutMode != "compact" {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "layoutMode", "invalid", "layoutMode is invalid")
	}
	if normalized.ThemeMode != "solid" && normalized.ThemeMode != "light" && normalized.ThemeMode != "dark" && normalized.ThemeMode != "minimal" {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "themeMode", "invalid", "themeMode is invalid")
	}
	if normalized.DefaultCurrency == "" {
		normalized.DefaultCurrency = "EUR"
	}
	if len(normalized.DefaultCurrency) != 3 {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "defaultCurrency", "invalid", "defaultCurrency must be a 3-letter code")
	}
	if normalized.Timezone == "" {
		normalized.Timezone = "Europe/Madrid"
	}
	if _, err := time.LoadLocation(normalized.Timezone); err != nil {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "timezone", "invalid", "timezone is invalid")
	}
	if normalized.TimerStillRunningHours <= 0 {
		normalized.TimerStillRunningHours = 8
	}
	if normalized.TimerStillRunningHours > 24 {
		return ProfileUpdateInput{}, validationError(ErrInvalidProfileInput, "timerStillRunningHours", "invalid", "timerStillRunningHours must be between 1 and 24")
	}

	return normalized, nil
}
