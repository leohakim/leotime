package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrInvoiceSeriesNotFound = errors.New("invoice series not found")
var ErrInvalidInvoiceSeriesInput = errors.New("invalid invoice series input")

var invoiceSeriesPatternToken = regexp.MustCompile(`\{YYYY\}|\{YY\}|\{SEQ(?::\d{1,2})?\}`)

type InvoiceSeries struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Pattern      string `json:"pattern"`
	NextSequence int    `json:"nextSequence"`
	ResetPolicy  string `json:"resetPolicy"`
	Active       bool   `json:"active"`
	Default      bool   `json:"default"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type InvoiceSeriesInput struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Pattern      string `json:"pattern"`
	ResetPolicy  string `json:"resetPolicy"`
	Active       *bool  `json:"active"`
	Default      *bool  `json:"default"`
	NextSequence *int   `json:"nextSequence"`
}

func (s *Store) ListInvoiceSeries(ctx context.Context, userID string) ([]InvoiceSeries, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, name, pattern, next_sequence, reset_policy, active, is_default, created_at, updated_at
		FROM invoice_series
		WHERE user_id = ?
		ORDER BY lower(name), created_at
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list invoice series: %w", err)
	}
	defer rows.Close()

	var seriesList []InvoiceSeries
	for rows.Next() {
		item, err := scanInvoiceSeries(rows)
		if err != nil {
			return nil, err
		}
		seriesList = append(seriesList, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoice series: %w", err)
	}
	return seriesList, nil
}

func (s *Store) InvoiceSeriesByID(ctx context.Context, userID, seriesID string) (*InvoiceSeries, error) {
	series, err := queryInvoiceSeries(ctx, s.db, `
		SELECT id, code, name, pattern, next_sequence, reset_policy, active, is_default, created_at, updated_at
		FROM invoice_series
		WHERE user_id = ? AND id = ?
	`, userID, seriesID)
	if err != nil {
		return nil, err
	}
	return series, nil
}

func (s *Store) CreateInvoiceSeries(ctx context.Context, userID string, input InvoiceSeriesInput) (*InvoiceSeries, error) {
	normalized, err := normalizeInvoiceSeriesInput(input, true)
	if err != nil {
		return nil, err
	}

	seriesID, err := newID("ser")
	if err != nil {
		return nil, err
	}
	now := nowString()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin invoice series create: %w", err)
	}
	defer tx.Rollback()

	if normalized.Default {
		if _, err := tx.ExecContext(ctx, `
			UPDATE invoice_series SET is_default = 0, updated_at = ? WHERE user_id = ?
		`, now, userID); err != nil {
			return nil, fmt.Errorf("clear default invoice series: %w", err)
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO invoice_series (
			id, user_id, code, name, pattern, next_sequence, reset_policy, active, is_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, seriesID, userID, normalized.Code, normalized.Name, normalized.Pattern, normalized.NextSequence,
		normalized.ResetPolicy, boolInt(normalized.Active), boolInt(normalized.Default), now, now)
	if err != nil {
		return nil, fmt.Errorf("insert invoice series: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice series create: %w", err)
	}
	return s.InvoiceSeriesByID(ctx, userID, seriesID)
}

func (s *Store) UpdateInvoiceSeries(ctx context.Context, userID, seriesID string, input InvoiceSeriesInput) (*InvoiceSeries, error) {
	current, err := s.InvoiceSeriesByID(ctx, userID, seriesID)
	if err != nil {
		return nil, err
	}

	normalized, err := normalizeInvoiceSeriesUpdate(current, input)
	if err != nil {
		return nil, err
	}

	now := nowString()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin invoice series update: %w", err)
	}
	defer tx.Rollback()

	if normalized.Default && !current.Default {
		if _, err := tx.ExecContext(ctx, `
			UPDATE invoice_series SET is_default = 0, updated_at = ? WHERE user_id = ?
		`, now, userID); err != nil {
			return nil, fmt.Errorf("clear default invoice series: %w", err)
		}
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE invoice_series
		SET code = ?, name = ?, pattern = ?, next_sequence = ?, reset_policy = ?,
			active = ?, is_default = ?, updated_at = ?
		WHERE user_id = ? AND id = ?
	`, normalized.Code, normalized.Name, normalized.Pattern, normalized.NextSequence, normalized.ResetPolicy,
		boolInt(normalized.Active), boolInt(normalized.Default), now, userID, seriesID)
	if err != nil {
		return nil, fmt.Errorf("update invoice series: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update invoice series rows: %w", err)
	}
	if rows == 0 {
		return nil, ErrInvoiceSeriesNotFound
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice series update: %w", err)
	}
	return s.InvoiceSeriesByID(ctx, userID, seriesID)
}

func FormatInvoiceNumber(pattern string, issueTime time.Time, sequence int) (string, error) {
	if err := validateInvoiceSeriesPattern(pattern); err != nil {
		return "", err
	}
	if sequence < 1 {
		return "", fmt.Errorf("sequence must be positive")
	}

	result := pattern
	result = strings.ReplaceAll(result, "{YYYY}", fmt.Sprintf("%04d", issueTime.Year()))
	result = strings.ReplaceAll(result, "{YY}", fmt.Sprintf("%02d", issueTime.Year()%100))

	seqToken := regexp.MustCompile(`\{SEQ(?::(\d{1,2}))?\}`)
	result = seqToken.ReplaceAllStringFunc(result, func(token string) string {
		matches := seqToken.FindStringSubmatch(token)
		if len(matches) == 2 && matches[1] != "" {
			width, _ := strconv.Atoi(matches[1])
			if width < 1 {
				width = 1
			}
			if width > 12 {
				width = 12
			}
			return fmt.Sprintf("%0*d", width, sequence)
		}
		return strconv.Itoa(sequence)
	})

	return result, nil
}

func (s *Store) NextInvoiceNumberTx(ctx context.Context, tx *sql.Tx, userID, seriesID string, issueTime time.Time) (string, int, error) {
	series, err := queryInvoiceSeries(ctx, tx, `
		SELECT id, code, name, pattern, next_sequence, reset_policy, active, is_default, created_at, updated_at
		FROM invoice_series
		WHERE user_id = ? AND id = ?
	`, userID, seriesID)
	if err != nil {
		return "", 0, err
	}
	if !series.Active {
		return "", 0, validationError(ErrInvalidInvoiceSeriesInput, "seriesId", "invalid", "invoice series is inactive")
	}

	sequence := series.NextSequence
	if series.ResetPolicy == "yearly" {
		var maxYear sql.NullInt64
		err = tx.QueryRowContext(ctx, `
			SELECT MAX(CAST(strftime('%Y', issued_at) AS INTEGER))
			FROM invoices
			WHERE user_id = ? AND series_id = ? AND issued_at IS NOT NULL
		`, userID, seriesID).Scan(&maxYear)
		if err != nil {
			return "", 0, fmt.Errorf("query invoice series year: %w", err)
		}
		issueYear := issueTime.Year()
		if maxYear.Valid && int(maxYear.Int64) < issueYear {
			sequence = 1
		}
	}

	number, err := FormatInvoiceNumber(series.Pattern, issueTime, sequence)
	if err != nil {
		return "", 0, err
	}

	now := nowString()
	_, err = tx.ExecContext(ctx, `
		UPDATE invoice_series SET next_sequence = ?, updated_at = ? WHERE user_id = ? AND id = ?
	`, sequence+1, now, userID, seriesID)
	if err != nil {
		return "", 0, fmt.Errorf("advance invoice series: %w", err)
	}

	return number, sequence, nil
}

type normalizedInvoiceSeries struct {
	Code         string
	Name         string
	Pattern      string
	NextSequence int
	ResetPolicy  string
	Active       bool
	Default      bool
}

func normalizeInvoiceSeriesInput(input InvoiceSeriesInput, isCreate bool) (normalizedInvoiceSeries, error) {
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if code == "" {
		return normalizedInvoiceSeries{}, validationError(ErrInvalidInvoiceSeriesInput, "code", "required", "code is required")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return normalizedInvoiceSeries{}, validationError(ErrInvalidInvoiceSeriesInput, "name", "required", "name is required")
	}

	pattern := strings.TrimSpace(input.Pattern)
	if pattern == "" {
		pattern = "{YYYY}-{SEQ:04}"
	}
	if err := validateInvoiceSeriesPattern(pattern); err != nil {
		return normalizedInvoiceSeries{}, validationError(ErrInvalidInvoiceSeriesInput, "pattern", "invalid", err.Error())
	}

	resetPolicy := strings.TrimSpace(strings.ToLower(input.ResetPolicy))
	if resetPolicy == "" {
		resetPolicy = "yearly"
	}
	if resetPolicy != "never" && resetPolicy != "yearly" {
		return normalizedInvoiceSeries{}, validationError(ErrInvalidInvoiceSeriesInput, "resetPolicy", "invalid", "resetPolicy must be never or yearly")
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}
	defaultSeries := false
	if input.Default != nil {
		defaultSeries = *input.Default
	}

	nextSequence := 1
	if input.NextSequence != nil {
		nextSequence = *input.NextSequence
	} else if !isCreate {
		nextSequence = 0
	}
	if isCreate && nextSequence < 1 {
		return normalizedInvoiceSeries{}, validationError(ErrInvalidInvoiceSeriesInput, "nextSequence", "invalid", "nextSequence must be at least 1")
	}
	if !isCreate && input.NextSequence != nil && nextSequence < 1 {
		return normalizedInvoiceSeries{}, validationError(ErrInvalidInvoiceSeriesInput, "nextSequence", "invalid", "nextSequence must be at least 1")
	}
	if !isCreate && input.NextSequence == nil {
		nextSequence = 1
	}

	return normalizedInvoiceSeries{
		Code:         code,
		Name:         name,
		Pattern:      pattern,
		NextSequence: nextSequence,
		ResetPolicy:  resetPolicy,
		Active:       active,
		Default:      defaultSeries,
	}, nil
}

func normalizeInvoiceSeriesUpdate(current *InvoiceSeries, input InvoiceSeriesInput) (normalizedInvoiceSeries, error) {
	code := current.Code
	if strings.TrimSpace(input.Code) != "" {
		code = strings.ToUpper(strings.TrimSpace(input.Code))
	}
	name := current.Name
	if strings.TrimSpace(input.Name) != "" {
		name = strings.TrimSpace(input.Name)
	}
	pattern := current.Pattern
	if strings.TrimSpace(input.Pattern) != "" {
		pattern = strings.TrimSpace(input.Pattern)
	}
	resetPolicy := current.ResetPolicy
	if strings.TrimSpace(input.ResetPolicy) != "" {
		resetPolicy = strings.TrimSpace(strings.ToLower(input.ResetPolicy))
	}
	active := current.Active
	if input.Active != nil {
		active = *input.Active
	}
	defaultSeries := current.Default
	if input.Default != nil {
		defaultSeries = *input.Default
	}
	nextSequence := current.NextSequence
	if input.NextSequence != nil {
		nextSequence = *input.NextSequence
	}

	normalized, err := normalizeInvoiceSeriesInput(InvoiceSeriesInput{
		Code:         code,
		Name:         name,
		Pattern:      pattern,
		ResetPolicy:  resetPolicy,
		Active:       &active,
		Default:      &defaultSeries,
		NextSequence: &nextSequence,
	}, false)
	if err != nil {
		return normalizedInvoiceSeries{}, err
	}
	return normalized, nil
}

func validateInvoiceSeriesPattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return fmt.Errorf("pattern is required")
	}
	if !invoiceSeriesPatternToken.MatchString(pattern) {
		return fmt.Errorf("pattern must include at least one supported placeholder")
	}
	remaining := invoiceSeriesPatternToken.ReplaceAllString(pattern, "")
	if strings.Contains(remaining, "{") || strings.Contains(remaining, "}") {
		return fmt.Errorf("pattern contains unsupported placeholders")
	}
	return nil
}

type invoiceSeriesScanner interface {
	Scan(dest ...any) error
}

func scanInvoiceSeries(scanner invoiceSeriesScanner) (InvoiceSeries, error) {
	var series InvoiceSeries
	var active, isDefault int
	if err := scanner.Scan(
		&series.ID, &series.Code, &series.Name, &series.Pattern, &series.NextSequence,
		&series.ResetPolicy, &active, &isDefault, &series.CreatedAt, &series.UpdatedAt,
	); err != nil {
		return InvoiceSeries{}, err
	}
	series.Active = active == 1
	series.Default = isDefault == 1
	return series, nil
}

func queryInvoiceSeries(ctx context.Context, db queryer, query string, args ...any) (*InvoiceSeries, error) {
	series, err := scanInvoiceSeries(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvoiceSeriesNotFound
		}
		return nil, fmt.Errorf("query invoice series: %w", err)
	}
	return &series, nil
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
