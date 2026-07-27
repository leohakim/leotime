package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var ErrInvalidVCSInput = errors.New("invalid vcs input")
var ErrVCSConnectionNotFound = errors.New("vcs connection not found")
var ErrVCSRepositoryNotFound = errors.New("vcs repository not found")

type VCSConnection struct {
	ID              string
	Provider        string
	BaseURL         string
	OwnerIdentity   string
	DefaultClientID string
	TokenConfigured bool
	Enabled         bool
	CreatedAt       string
	UpdatedAt       string
}

type VCSConnectionInput struct {
	Provider        string
	BaseURL         string
	OwnerIdentity   string
	DefaultClientID string
}

type VCSRepository struct {
	ID                string
	ConnectionID      string
	Owner             string
	Name              string
	ClientID          string
	EffectiveClientID string
	ProjectID         string
	Enabled           bool
	CreatedAt         string
	UpdatedAt         string
}

type VCSRepositoryInput struct {
	ConnectionID string
	Owner        string
	Name         string
	ClientID     string
	ProjectID    string
}

type vcsConnectionRecord struct {
	VCSConnection
	TokenEnc string
}

type VCSCollectionLink struct {
	Connection VCSConnection
	Repository VCSRepository
	TokenEnc   string
}

func (s *Store) CreateVCSConnection(ctx context.Context, userID string, input VCSConnectionInput, tokenEnc string) (*VCSConnection, error) {
	normalized, err := s.normalizeVCSConnectionInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}
	id, err := newID("vcs")
	if err != nil {
		return nil, err
	}
	now := nowString()
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO vcs_connections (id, user_id, provider, base_url, owner_identity, token_enc, default_client_id, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)",
		id, userID, normalized.Provider, normalized.BaseURL, normalized.OwnerIdentity, strings.TrimSpace(tokenEnc), nullValue(normalized.DefaultClientID), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert vcs connection: %w", err)
	}
	return s.VCSConnectionByID(ctx, userID, id)
}

func (s *Store) VCSConnectionByID(ctx context.Context, userID, connectionID string) (*VCSConnection, error) {
	record, err := s.vcsConnectionRecordByID(ctx, userID, connectionID)
	if err != nil {
		return nil, err
	}
	return &record.VCSConnection, nil
}

func (s *Store) VCSConnectionCredential(ctx context.Context, userID, connectionID string) (baseURL, ownerIdentity, tokenEnc string, err error) {
	record, err := s.vcsConnectionRecordByID(ctx, userID, connectionID)
	if err != nil {
		return "", "", "", err
	}
	return record.BaseURL, record.OwnerIdentity, record.TokenEnc, nil
}

func (s *Store) ListVCSConnections(ctx context.Context, userID string) ([]VCSConnection, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, provider, base_url, owner_identity, default_client_id, token_enc, enabled, created_at, updated_at FROM vcs_connections WHERE user_id = ? ORDER BY lower(base_url), created_at",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list vcs connections: %w", err)
	}
	defer rows.Close()
	connections := make([]VCSConnection, 0)
	for rows.Next() {
		record, err := scanVCSConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, record.VCSConnection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vcs connections: %w", err)
	}
	return connections, nil
}

func (s *Store) ListVCSRepositories(ctx context.Context, userID string) ([]VCSRepository, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT r.id, r.connection_id, r.owner, r.name, r.client_id, COALESCE(r.client_id, c.default_client_id, ''), r.project_id, r.enabled, r.created_at, r.updated_at FROM vcs_repositories r JOIN vcs_connections c ON c.id = r.connection_id AND c.user_id = r.user_id WHERE r.user_id = ? ORDER BY lower(r.owner), lower(r.name)",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list vcs repositories: %w", err)
	}
	defer rows.Close()
	repositories := make([]VCSRepository, 0)
	for rows.Next() {
		repository, err := scanVCSRepository(rows)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, repository)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vcs repositories: %w", err)
	}
	return repositories, nil
}

func (s *Store) ListVCSCollectionLinks(ctx context.Context, userID, clientID, projectID string) ([]VCSCollectionLink, error) {
	clientID = strings.TrimSpace(clientID)
	projectID = strings.TrimSpace(projectID)
	if clientID == "" {
		return nil, nil
	}
	query := "SELECT r.id, r.connection_id, r.owner, r.name, r.client_id, COALESCE(r.client_id, c.default_client_id, ''), r.project_id, r.enabled, r.created_at, r.updated_at, c.id, c.provider, c.base_url, c.owner_identity, c.default_client_id, c.token_enc, c.enabled, c.created_at, c.updated_at FROM vcs_repositories r JOIN vcs_connections c ON c.id = r.connection_id AND c.user_id = r.user_id WHERE r.user_id = ? AND r.enabled = 1 AND c.enabled = 1 AND COALESCE(r.client_id, c.default_client_id, '') = ?"
	args := []any{userID, clientID}
	if projectID != "" {
		query += " AND r.project_id = ?"
		args = append(args, projectID)
	}
	query += " ORDER BY lower(r.owner), lower(r.name)"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list vcs collection links: %w", err)
	}
	defer rows.Close()
	links := make([]VCSCollectionLink, 0)
	for rows.Next() {
		repository, connection, err := scanVCSCollectionLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, VCSCollectionLink{Connection: connection.VCSConnection, Repository: repository, TokenEnc: connection.TokenEnc})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vcs collection links: %w", err)
	}
	return links, nil
}

func (s *Store) DeleteVCSConnection(ctx context.Context, userID, connectionID string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM vcs_connections WHERE user_id = ? AND id = ?", userID, connectionID)
	if err != nil {
		return fmt.Errorf("delete vcs connection: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect vcs connection delete: %w", err)
	}
	if affected == 0 {
		return ErrVCSConnectionNotFound
	}
	return nil
}

func (s *Store) UpdateVCSConnection(ctx context.Context, userID, connectionID string, input VCSConnectionInput, tokenEnc string) (*VCSConnection, error) {
	existing, err := s.vcsConnectionRecordByID(ctx, userID, connectionID)
	if err != nil {
		return nil, err
	}
	normalized, err := s.normalizeVCSConnectionInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}
	nextToken := strings.TrimSpace(tokenEnc)
	if nextToken == "" {
		nextToken = existing.TokenEnc
	}
	now := nowString()
	result, err := s.db.ExecContext(ctx,
		"UPDATE vcs_connections SET provider = ?, base_url = ?, owner_identity = ?, token_enc = ?, default_client_id = ?, updated_at = ? WHERE user_id = ? AND id = ?",
		normalized.Provider, normalized.BaseURL, normalized.OwnerIdentity, nextToken, nullValue(normalized.DefaultClientID), now, userID, connectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("update vcs connection: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect vcs connection update: %w", err)
	}
	if affected == 0 {
		return nil, ErrVCSConnectionNotFound
	}
	return s.VCSConnectionByID(ctx, userID, connectionID)
}

func (s *Store) UpsertVCSRepository(ctx context.Context, userID string, input VCSRepositoryInput) (*VCSRepository, error) {
	normalized, effectiveClientID, err := s.normalizeVCSRepositoryInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}
	id, err := newID("vcr")
	if err != nil {
		return nil, err
	}
	now := nowString()
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO vcs_repositories (id, user_id, connection_id, owner, name, client_id, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, userID, normalized.ConnectionID, normalized.Owner, normalized.Name, nullValue(normalized.ClientID), nullValue(normalized.ProjectID), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert vcs repository: %w", err)
	}
	repository, err := s.VCSRepositoryByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if repository.EffectiveClientID != effectiveClientID {
		return nil, fmt.Errorf("load effective client: %w", ErrInvalidVCSInput)
	}
	return repository, nil
}

func (s *Store) VCSRepositoryByID(ctx context.Context, userID, repositoryID string) (*VCSRepository, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT r.id, r.connection_id, r.owner, r.name, r.client_id, COALESCE(r.client_id, c.default_client_id, ''), r.project_id, r.enabled, r.created_at, r.updated_at FROM vcs_repositories r JOIN vcs_connections c ON c.id = r.connection_id AND c.user_id = r.user_id WHERE r.user_id = ? AND r.id = ?",
		userID, repositoryID,
	)
	var repository VCSRepository
	var clientID, effectiveClientID, projectID sql.NullString
	var enabled int
	err := row.Scan(
		&repository.ID, &repository.ConnectionID, &repository.Owner, &repository.Name,
		&clientID, &effectiveClientID, &projectID, &enabled, &repository.CreatedAt, &repository.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrVCSRepositoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query vcs repository: %w", err)
	}
	repository.ClientID = clientID.String
	repository.EffectiveClientID = effectiveClientID.String
	repository.ProjectID = projectID.String
	repository.Enabled = enabled != 0
	return &repository, nil
}

func (s *Store) UpdateVCSRepository(ctx context.Context, userID, repositoryID string, input VCSRepositoryInput) (*VCSRepository, error) {
	if _, err := s.VCSRepositoryByID(ctx, userID, repositoryID); err != nil {
		return nil, err
	}
	normalized, effectiveClientID, err := s.normalizeVCSRepositoryInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}
	now := nowString()
	result, err := s.db.ExecContext(ctx,
		"UPDATE vcs_repositories SET connection_id = ?, owner = ?, name = ?, client_id = ?, project_id = ?, updated_at = ? WHERE user_id = ? AND id = ?",
		normalized.ConnectionID, normalized.Owner, normalized.Name, nullValue(normalized.ClientID), nullValue(normalized.ProjectID), now, userID, repositoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("update vcs repository: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect vcs repository update: %w", err)
	}
	if affected == 0 {
		return nil, ErrVCSRepositoryNotFound
	}
	repository, err := s.VCSRepositoryByID(ctx, userID, repositoryID)
	if err != nil {
		return nil, err
	}
	if repository.EffectiveClientID != effectiveClientID {
		return nil, fmt.Errorf("load effective client: %w", ErrInvalidVCSInput)
	}
	return repository, nil
}

func (s *Store) DeleteVCSRepository(ctx context.Context, userID, repositoryID string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM vcs_repositories WHERE user_id = ? AND id = ?", userID, repositoryID)
	if err != nil {
		return fmt.Errorf("delete vcs repository: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect vcs repository delete: %w", err)
	}
	if affected == 0 {
		return ErrVCSRepositoryNotFound
	}
	return nil
}

func (s *Store) vcsConnectionRecordByID(ctx context.Context, userID, connectionID string) (*vcsConnectionRecord, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, provider, base_url, owner_identity, default_client_id, token_enc, enabled, created_at, updated_at FROM vcs_connections WHERE user_id = ? AND id = ?",
		userID, connectionID,
	)
	var record vcsConnectionRecord
	var defaultClientID sql.NullString
	var enabled int
	err := row.Scan(
		&record.ID, &record.Provider, &record.BaseURL, &record.OwnerIdentity,
		&defaultClientID, &record.TokenEnc, &enabled, &record.CreatedAt, &record.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrVCSConnectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query vcs connection: %w", err)
	}
	record.DefaultClientID = defaultClientID.String
	record.TokenConfigured = strings.TrimSpace(record.TokenEnc) != ""
	record.Enabled = enabled != 0
	return &record, nil
}

type vcsConnectionScanner interface {
	Scan(dest ...any) error
}

func scanVCSConnection(scanner vcsConnectionScanner) (vcsConnectionRecord, error) {
	var record vcsConnectionRecord
	var defaultClientID sql.NullString
	var enabled int
	if err := scanner.Scan(
		&record.ID, &record.Provider, &record.BaseURL, &record.OwnerIdentity,
		&defaultClientID, &record.TokenEnc, &enabled, &record.CreatedAt, &record.UpdatedAt,
	); err != nil {
		return vcsConnectionRecord{}, fmt.Errorf("scan vcs connection: %w", err)
	}
	record.DefaultClientID = defaultClientID.String
	record.TokenConfigured = strings.TrimSpace(record.TokenEnc) != ""
	record.Enabled = enabled != 0
	return record, nil
}

type vcsRepositoryScanner interface {
	Scan(dest ...any) error
}

func scanVCSRepository(scanner vcsRepositoryScanner) (VCSRepository, error) {
	var repository VCSRepository
	var clientID, effectiveClientID, projectID sql.NullString
	var enabled int
	if err := scanner.Scan(
		&repository.ID, &repository.ConnectionID, &repository.Owner, &repository.Name,
		&clientID, &effectiveClientID, &projectID, &enabled, &repository.CreatedAt, &repository.UpdatedAt,
	); err != nil {
		return VCSRepository{}, fmt.Errorf("scan vcs repository: %w", err)
	}
	repository.ClientID = clientID.String
	repository.EffectiveClientID = effectiveClientID.String
	repository.ProjectID = projectID.String
	repository.Enabled = enabled != 0
	return repository, nil
}

func scanVCSCollectionLink(scanner vcsRepositoryScanner) (VCSRepository, vcsConnectionRecord, error) {
	var repository VCSRepository
	var connection vcsConnectionRecord
	var clientID, effectiveClientID, projectID, defaultClientID sql.NullString
	var repositoryEnabled, connectionEnabled int
	if err := scanner.Scan(
		&repository.ID, &repository.ConnectionID, &repository.Owner, &repository.Name,
		&clientID, &effectiveClientID, &projectID, &repositoryEnabled, &repository.CreatedAt, &repository.UpdatedAt,
		&connection.ID, &connection.Provider, &connection.BaseURL, &connection.OwnerIdentity,
		&defaultClientID, &connection.TokenEnc, &connectionEnabled, &connection.CreatedAt, &connection.UpdatedAt,
	); err != nil {
		return VCSRepository{}, vcsConnectionRecord{}, fmt.Errorf("scan vcs collection link: %w", err)
	}
	repository.ClientID = clientID.String
	repository.EffectiveClientID = effectiveClientID.String
	repository.ProjectID = projectID.String
	repository.Enabled = repositoryEnabled != 0
	connection.DefaultClientID = defaultClientID.String
	connection.TokenConfigured = strings.TrimSpace(connection.TokenEnc) != ""
	connection.Enabled = connectionEnabled != 0
	return repository, connection, nil
}

func (s *Store) normalizeVCSConnectionInput(ctx context.Context, userID string, input VCSConnectionInput) (VCSConnectionInput, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.OwnerIdentity = strings.TrimSpace(input.OwnerIdentity)
	input.DefaultClientID = strings.TrimSpace(input.DefaultClientID)
	baseURL, err := normalizeVCSBaseURL(input.BaseURL)
	if err != nil {
		return VCSConnectionInput{}, validationError(ErrInvalidVCSInput, "baseUrl", "invalid", err.Error())
	}
	input.BaseURL = baseURL
	if input.Provider != "gitea" {
		return VCSConnectionInput{}, validationError(ErrInvalidVCSInput, "provider", "unsupported", "provider must be gitea")
	}
	if input.DefaultClientID != "" {
		exists, err := s.activeClientExists(ctx, userID, input.DefaultClientID)
		if err != nil {
			return VCSConnectionInput{}, err
		}
		if !exists {
			return VCSConnectionInput{}, validationError(ErrInvalidVCSInput, "defaultClientId", "invalid", "defaultClientId must reference an active client")
		}
	}
	return input, nil
}

func (s *Store) normalizeVCSRepositoryInput(ctx context.Context, userID string, input VCSRepositoryInput) (VCSRepositoryInput, string, error) {
	input.ConnectionID = strings.TrimSpace(input.ConnectionID)
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	owner, name, err := normalizeVCSRepoOwnerName(input.Owner, input.Name)
	if err != nil {
		return VCSRepositoryInput{}, "", err
	}
	input.Owner, input.Name = owner, name
	if input.ConnectionID == "" || input.Owner == "" || input.Name == "" {
		return VCSRepositoryInput{}, "", validationError(ErrInvalidVCSInput, "repository", "required", "connectionId, owner, and name are required")
	}
	connection, err := s.vcsConnectionRecordByID(ctx, userID, input.ConnectionID)
	if errors.Is(err, ErrVCSConnectionNotFound) {
		return VCSRepositoryInput{}, "", validationError(ErrInvalidVCSInput, "connectionId", "invalid", "connectionId must reference a connection")
	}
	if err != nil {
		return VCSRepositoryInput{}, "", err
	}
	effectiveClientID := input.ClientID
	if effectiveClientID == "" {
		effectiveClientID = connection.DefaultClientID
	}
	if effectiveClientID == "" {
		return VCSRepositoryInput{}, "", validationError(ErrInvalidVCSInput, "clientId", "required", "repository must resolve to one client")
	}
	exists, err := s.activeClientExists(ctx, userID, effectiveClientID)
	if err != nil {
		return VCSRepositoryInput{}, "", err
	}
	if !exists {
		return VCSRepositoryInput{}, "", validationError(ErrInvalidVCSInput, "clientId", "invalid", "clientId must reference an active client")
	}
	if input.ProjectID != "" {
		var projectClientID sql.NullString
		err := s.db.QueryRowContext(ctx,
			"SELECT client_id FROM projects WHERE user_id = ? AND id = ? AND archived_at IS NULL",
			userID, input.ProjectID,
		).Scan(&projectClientID)
		if errors.Is(err, sql.ErrNoRows) || projectClientID.String != effectiveClientID {
			return VCSRepositoryInput{}, "", validationError(ErrInvalidVCSInput, "projectId", "invalid", "projectId must belong to the repository client")
		}
		if err != nil {
			return VCSRepositoryInput{}, "", fmt.Errorf("check vcs repository project: %w", err)
		}
	}
	return input, effectiveClientID, nil
}

func normalizeVCSRepoOwnerName(owner, name string) (string, string, error) {
	owner = strings.Trim(strings.TrimSpace(owner), "/")
	name = strings.Trim(strings.TrimSpace(name), "/")

	// Allow pasting org/repo or host/org/repo into the name (or owner) field.
	candidate := ""
	switch {
	case strings.Contains(name, "/"):
		candidate = name
	case strings.Contains(owner, "/") && (name == "" || strings.Contains(owner, "@")):
		candidate = owner
	}
	if candidate != "" {
		parts := strings.Split(candidate, "/")
		for len(parts) > 2 && strings.Contains(parts[0], ".") {
			parts = parts[1:]
		}
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", validationError(ErrInvalidVCSInput, "name", "invalid", "repository must be org/name, e.g. ENACT/backend")
		}
		owner, name = parts[0], parts[1]
	}

	if strings.Contains(owner, "@") {
		return "", "", validationError(ErrInvalidVCSInput, "owner", "invalid", "owner must be the Gitea org or username, not an email")
	}
	if strings.Contains(owner, "/") || strings.Contains(name, "/") {
		return "", "", validationError(ErrInvalidVCSInput, "repository", "invalid", "use owner=ENACT and name=backend (not a URL path)")
	}
	if owner == "" || name == "" {
		return "", "", validationError(ErrInvalidVCSInput, "repository", "required", "connectionId, owner, and name are required")
	}
	return owner, name, nil
}

func normalizeVCSBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("baseUrl must be a valid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("baseUrl must use http or https")
	}
	if parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("baseUrl must contain only scheme and host")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("baseUrl must not contain a path")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", errors.New("baseUrl host is required")
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()) {
		return "", errors.New("baseUrl host is not allowed")
	}
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = ""
	return strings.TrimSuffix(parsed.String(), "/"), nil
}
