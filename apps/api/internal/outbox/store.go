package outbox

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrDuplicate = errors.New("outbox entry already exists")
var ErrNotFound = errors.New("outbox entry not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Enqueue(ctx context.Context, input EnqueueInput, now time.Time) (*Email, error) {
	normalized, err := normalizeEnqueueInput(input)
	if err != nil {
		return nil, err
	}

	id, err := newID("eml")
	if err != nil {
		return nil, err
	}

	nowString := formatTime(now)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO email_outbox (
			id, user_id, time_entry_id, kind, to_address, subject, body_text,
			status, attempts, max_attempts, next_retry_at, last_error, sent_at,
			created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', 0, ?, ?, '', NULL, ?, ?)
	`, id, normalized.UserID, nullString(normalized.TimeEntryID), normalized.Kind, normalized.ToAddress,
		normalized.Subject, normalized.BodyText, normalized.MaxAttempts, nowString, "", nowString, nowString)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("insert outbox email: %w", err)
	}

	return s.ByID(ctx, id)
}

func (s *Store) ListDuePending(ctx context.Context, limit int, now time.Time) ([]Email, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, COALESCE(time_entry_id, ''), kind, to_address, subject, body_text,
			status, attempts, max_attempts, next_retry_at, last_error, COALESCE(sent_at, ''),
			created_at, updated_at
		FROM email_outbox
		WHERE status = 'pending' AND next_retry_at <= ?
		ORDER BY next_retry_at ASC, created_at ASC
		LIMIT ?
	`, formatTime(now), limit)
	if err != nil {
		return nil, fmt.Errorf("list due pending outbox emails: %w", err)
	}
	defer rows.Close()

	var emails []Email
	for rows.Next() {
		email, err := scanEmail(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due pending outbox emails: %w", err)
	}
	return emails, nil
}

func (s *Store) ByID(ctx context.Context, id string) (*Email, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, COALESCE(time_entry_id, ''), kind, to_address, subject, body_text,
			status, attempts, max_attempts, next_retry_at, last_error, COALESCE(sent_at, ''),
			created_at, updated_at
		FROM email_outbox
		WHERE id = ?
	`, id)

	email, err := scanEmailRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query outbox email: %w", err)
	}
	return &email, nil
}

func (s *Store) MarkSent(ctx context.Context, id string, sentAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE email_outbox
		SET status = 'sent', sent_at = ?, updated_at = ?, last_error = ''
		WHERE id = ? AND status = 'pending'
	`, formatTime(sentAt), formatTime(sentAt), id)
	if err != nil {
		return fmt.Errorf("mark outbox email sent: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect mark sent result: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ScheduleRetry(ctx context.Context, id string, attempts int, nextRetryAt time.Time, lastError string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE email_outbox
		SET attempts = ?, next_retry_at = ?, last_error = ?, updated_at = ?
		WHERE id = ? AND status = 'pending'
	`, attempts, formatTime(nextRetryAt), truncateError(lastError), formatTime(nextRetryAt), id)
	if err != nil {
		return fmt.Errorf("schedule outbox retry: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect schedule retry result: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) MarkDead(ctx context.Context, id string, attempts int, lastError string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE email_outbox
		SET status = 'dead', attempts = ?, last_error = ?, updated_at = ?
		WHERE id = ? AND status = 'pending'
	`, attempts, truncateError(lastError), formatTime(now), id)
	if err != nil {
		return fmt.Errorf("mark outbox email dead: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect mark dead result: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func normalizeEnqueueInput(input EnqueueInput) (EnqueueInput, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.TimeEntryID = strings.TrimSpace(input.TimeEntryID)
	input.Kind = strings.TrimSpace(input.Kind)
	input.ToAddress = strings.TrimSpace(strings.ToLower(input.ToAddress))
	input.Subject = strings.TrimSpace(input.Subject)
	input.BodyText = strings.TrimSpace(input.BodyText)

	if input.UserID == "" {
		return EnqueueInput{}, fmt.Errorf("user id is required")
	}
	if input.Kind == "" {
		return EnqueueInput{}, fmt.Errorf("kind is required")
	}
	if input.ToAddress == "" {
		return EnqueueInput{}, fmt.Errorf("to address is required")
	}
	if input.Subject == "" {
		return EnqueueInput{}, fmt.Errorf("subject is required")
	}
	if input.BodyText == "" {
		return EnqueueInput{}, fmt.Errorf("body text is required")
	}
	if input.MaxAttempts <= 0 {
		input.MaxAttempts = 5
	}
	return input, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEmail(scanner rowScanner) (Email, error) {
	return scanEmailRow(scanner)
}

func scanEmailRow(scanner rowScanner) (Email, error) {
	var email Email
	if err := scanner.Scan(
		&email.ID,
		&email.UserID,
		&email.TimeEntryID,
		&email.Kind,
		&email.ToAddress,
		&email.Subject,
		&email.BodyText,
		&email.Status,
		&email.Attempts,
		&email.MaxAttempts,
		&email.NextRetryAt,
		&email.LastError,
		&email.SentAt,
		&email.CreatedAt,
		&email.UpdatedAt,
	); err != nil {
		return Email{}, err
	}
	return email, nil
}

func newID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func truncateError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 500 {
		return value
	}
	return value[:500]
}
