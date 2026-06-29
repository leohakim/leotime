package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

var ErrClientNotFound = errors.New("client not found")
var ErrInvalidClientInput = errors.New("invalid client input")

type Client struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Email                  string `json:"email"`
	TaxID                  string `json:"taxId"`
	BillingAddress         string `json:"billingAddress"`
	DefaultCurrency        string `json:"defaultCurrency"`
	DefaultHourlyRateMinor int64  `json:"defaultHourlyRateMinor"`
	ArchivedAt             string `json:"archivedAt"`
	CreatedAt              string `json:"createdAt"`
	UpdatedAt              string `json:"updatedAt"`
}

type ClientInput struct {
	Name                   string `json:"name"`
	Email                  string `json:"email"`
	TaxID                  string `json:"taxId"`
	BillingAddress         string `json:"billingAddress"`
	DefaultCurrency        string `json:"defaultCurrency"`
	DefaultHourlyRateMinor int64  `json:"defaultHourlyRateMinor"`
}

func (s *Store) ListClients(ctx context.Context, userID string, includeArchived bool) ([]Client, error) {
	query := `
		SELECT id, name, email, tax_id, billing_address, default_currency, default_hourly_rate_minor,
			archived_at, created_at, updated_at
		FROM clients
		WHERE user_id = ?
	`
	if !includeArchived {
		query += " AND archived_at IS NULL"
	}
	query += " ORDER BY lower(name), created_at"

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		client, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clients: %w", err)
	}
	return clients, nil
}

func (s *Store) ClientByID(ctx context.Context, userID string, clientID string) (*Client, error) {
	client, err := queryClient(ctx, s.db, `
		SELECT id, name, email, tax_id, billing_address, default_currency, default_hourly_rate_minor,
			archived_at, created_at, updated_at
		FROM clients
		WHERE user_id = ? AND id = ?
	`, userID, clientID)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (s *Store) CreateClient(ctx context.Context, userID string, input ClientInput) (*Client, error) {
	normalized, err := normalizeClientInput(input)
	if err != nil {
		return nil, err
	}

	clientID, err := newID("cli")
	if err != nil {
		return nil, err
	}
	now := nowString()

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO clients (
			id, user_id, name, email, tax_id, billing_address, default_currency,
			default_hourly_rate_minor, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, clientID, userID, normalized.Name, nullValue(normalized.Email), nullValue(normalized.TaxID),
		nullValue(normalized.BillingAddress), normalized.DefaultCurrency, normalized.DefaultHourlyRateMinor, now, now); err != nil {
		return nil, fmt.Errorf("insert client: %w", err)
	}

	return s.ClientByID(ctx, userID, clientID)
}

func (s *Store) UpdateClient(ctx context.Context, userID string, clientID string, input ClientInput) (*Client, error) {
	normalized, err := normalizeClientInput(input)
	if err != nil {
		return nil, err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE clients
		SET name = ?, email = ?, tax_id = ?, billing_address = ?, default_currency = ?,
			default_hourly_rate_minor = ?, updated_at = ?
		WHERE user_id = ? AND id = ?
	`, normalized.Name, nullValue(normalized.Email), nullValue(normalized.TaxID), nullValue(normalized.BillingAddress),
		normalized.DefaultCurrency, normalized.DefaultHourlyRateMinor, nowString(), userID, clientID)
	if err != nil {
		return nil, fmt.Errorf("update client: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect update client result: %w", err)
	}
	if affected == 0 {
		return nil, ErrClientNotFound
	}

	return s.ClientByID(ctx, userID, clientID)
}

func (s *Store) ArchiveClient(ctx context.Context, userID string, clientID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE clients
		SET archived_at = COALESCE(archived_at, ?), updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nowString(), nowString(), userID, clientID)
	if err != nil {
		return fmt.Errorf("archive client: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect archive client result: %w", err)
	}
	if affected == 0 {
		return ErrClientNotFound
	}
	return nil
}

type clientScanner interface {
	Scan(dest ...any) error
}

func queryClient(ctx context.Context, db *sql.DB, query string, args ...any) (*Client, error) {
	client, err := scanClient(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

func scanClient(scanner clientScanner) (Client, error) {
	var client Client
	var email sql.NullString
	var taxID sql.NullString
	var billingAddress sql.NullString
	var archivedAt sql.NullString

	if err := scanner.Scan(
		&client.ID,
		&client.Name,
		&email,
		&taxID,
		&billingAddress,
		&client.DefaultCurrency,
		&client.DefaultHourlyRateMinor,
		&archivedAt,
		&client.CreatedAt,
		&client.UpdatedAt,
	); err != nil {
		return Client{}, fmt.Errorf("scan client: %w", err)
	}

	client.Email = email.String
	client.TaxID = taxID.String
	client.BillingAddress = billingAddress.String
	client.ArchivedAt = archivedAt.String
	return client, nil
}

func normalizeClientInput(input ClientInput) (ClientInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.TaxID = strings.TrimSpace(input.TaxID)
	input.BillingAddress = strings.TrimSpace(input.BillingAddress)
	input.DefaultCurrency = strings.ToUpper(strings.TrimSpace(input.DefaultCurrency))
	if input.DefaultCurrency == "" {
		input.DefaultCurrency = "EUR"
	}

	if input.Name == "" {
		return ClientInput{}, fmt.Errorf("%w: name is required", ErrInvalidClientInput)
	}
	if !validCurrency(input.DefaultCurrency) {
		return ClientInput{}, fmt.Errorf("%w: defaultCurrency must be a 3-letter code", ErrInvalidClientInput)
	}
	if input.Email != "" && !validEmail(input.Email) {
		return ClientInput{}, fmt.Errorf("%w: email must be valid", ErrInvalidClientInput)
	}
	if input.DefaultHourlyRateMinor < 0 {
		return ClientInput{}, fmt.Errorf("%w: defaultHourlyRateMinor must be non-negative", ErrInvalidClientInput)
	}

	return input, nil
}

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}
	return true
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return false
	}
	parts := strings.Split(address.Address, "@")
	return len(parts) == 2 && strings.Contains(parts[1], ".")
}

func nullValue(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
