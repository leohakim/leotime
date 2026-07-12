package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrProjectNotFound = errors.New("project not found")
var ErrInvalidProjectInput = errors.New("invalid project input")

type Project struct {
	ID                     string `json:"id"`
	ClientID               string `json:"clientId"`
	ClientName             string `json:"clientName"`
	Name                   string `json:"name"`
	Color                  string `json:"color"`
	DefaultHourlyRateMinor *int64 `json:"defaultHourlyRateMinor"`
	LocalRepoPath          string `json:"localRepoPath"`
	GitRemoteURL           string `json:"gitRemoteUrl"`
	CursorWorkspaceSlug    string `json:"cursorWorkspaceSlug"`
	ArchivedAt             string `json:"archivedAt"`
	CreatedAt              string `json:"createdAt"`
	UpdatedAt              string `json:"updatedAt"`
}

type ProjectInput struct {
	ClientID               string `json:"clientId"`
	Name                   string `json:"name"`
	Color                  string `json:"color"`
	DefaultHourlyRateMinor *int64 `json:"defaultHourlyRateMinor"`
	LocalRepoPath          string `json:"localRepoPath"`
	GitRemoteURL           string `json:"gitRemoteUrl"`
	CursorWorkspaceSlug    string `json:"cursorWorkspaceSlug"`
}

func (s *Store) ListProjects(ctx context.Context, userID string, includeArchived bool, clientID string) ([]Project, error) {
	query := `
		SELECT p.id, p.client_id, COALESCE(c.name, ''), p.name, p.color, p.default_hourly_rate_minor,
			p.local_repo_path, p.git_remote_url, p.cursor_workspace_slug,
			p.archived_at, p.created_at, p.updated_at
		FROM projects p
		LEFT JOIN clients c ON c.id = p.client_id AND c.user_id = p.user_id
		WHERE p.user_id = ?
	`
	args := []any{userID}
	if !includeArchived {
		query += " AND p.archived_at IS NULL"
	}
	if strings.TrimSpace(clientID) != "" {
		query += " AND p.client_id = ?"
		args = append(args, strings.TrimSpace(clientID))
	}
	query += " ORDER BY lower(p.name), p.created_at"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

func (s *Store) ProjectByID(ctx context.Context, userID string, projectID string) (*Project, error) {
	project, err := queryProject(ctx, s.db, `
		SELECT p.id, p.client_id, COALESCE(c.name, ''), p.name, p.color, p.default_hourly_rate_minor,
			p.local_repo_path, p.git_remote_url, p.cursor_workspace_slug,
			p.archived_at, p.created_at, p.updated_at
		FROM projects p
		LEFT JOIN clients c ON c.id = p.client_id AND c.user_id = p.user_id
		WHERE p.user_id = ? AND p.id = ?
	`, userID, projectID)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (s *Store) CreateProject(ctx context.Context, userID string, input ProjectInput) (*Project, error) {
	normalized, err := s.normalizeProjectInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}

	projectID, err := newID("prj")
	if err != nil {
		return nil, err
	}
	now := nowString()

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO projects (
			id, user_id, client_id, name, color, default_hourly_rate_minor,
			local_repo_path, git_remote_url, cursor_workspace_slug,
			created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, projectID, userID, nullValue(normalized.ClientID), normalized.Name, normalized.Color,
		nullableInt64(normalized.DefaultHourlyRateMinor),
		normalized.LocalRepoPath, normalized.GitRemoteURL, normalized.CursorWorkspaceSlug,
		now, now); err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}

	return s.ProjectByID(ctx, userID, projectID)
}

func (s *Store) UpdateProject(ctx context.Context, userID string, projectID string, input ProjectInput) (*Project, error) {
	normalized, err := s.normalizeProjectInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE projects
		SET client_id = ?, name = ?, color = ?, default_hourly_rate_minor = ?,
			local_repo_path = ?, git_remote_url = ?, cursor_workspace_slug = ?,
			updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nullValue(normalized.ClientID), normalized.Name, normalized.Color,
		nullableInt64(normalized.DefaultHourlyRateMinor),
		normalized.LocalRepoPath, normalized.GitRemoteURL, normalized.CursorWorkspaceSlug,
		nowString(), userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect update project result: %w", err)
	}
	if affected == 0 {
		return nil, ErrProjectNotFound
	}

	return s.ProjectByID(ctx, userID, projectID)
}

func (s *Store) ArchiveProject(ctx context.Context, userID string, projectID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE projects
		SET archived_at = COALESCE(archived_at, ?), updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nowString(), nowString(), userID, projectID)
	if err != nil {
		return fmt.Errorf("archive project: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect archive project result: %w", err)
	}
	if affected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (s *Store) RestoreProject(ctx context.Context, userID string, projectID string) (*Project, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE projects
		SET archived_at = NULL, updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nowString(), userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("restore project: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect restore project result: %w", err)
	}
	if affected == 0 {
		return nil, ErrProjectNotFound
	}

	return s.ProjectByID(ctx, userID, projectID)
}

type projectScanner interface {
	Scan(dest ...any) error
}

func queryProject(ctx context.Context, db *sql.DB, query string, args ...any) (*Project, error) {
	project, err := scanProject(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}

func scanProject(scanner projectScanner) (Project, error) {
	var project Project
	var clientID sql.NullString
	var defaultHourlyRateMinor sql.NullInt64
	var archivedAt sql.NullString

	if err := scanner.Scan(
		&project.ID,
		&clientID,
		&project.ClientName,
		&project.Name,
		&project.Color,
		&defaultHourlyRateMinor,
		&project.LocalRepoPath,
		&project.GitRemoteURL,
		&project.CursorWorkspaceSlug,
		&archivedAt,
		&project.CreatedAt,
		&project.UpdatedAt,
	); err != nil {
		return Project{}, fmt.Errorf("scan project: %w", err)
	}

	project.ClientID = clientID.String
	if defaultHourlyRateMinor.Valid {
		project.DefaultHourlyRateMinor = &defaultHourlyRateMinor.Int64
	}
	project.ArchivedAt = archivedAt.String
	return project, nil
}

func (s *Store) normalizeProjectInput(ctx context.Context, userID string, input ProjectInput) (ProjectInput, error) {
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.Name = strings.TrimSpace(input.Name)
	input.Color = strings.TrimSpace(input.Color)
	input.LocalRepoPath = strings.TrimSpace(input.LocalRepoPath)
	input.GitRemoteURL = strings.TrimSpace(input.GitRemoteURL)
	input.CursorWorkspaceSlug = strings.TrimSpace(input.CursorWorkspaceSlug)
	if input.Color == "" {
		input.Color = "#2563eb"
	}

	if input.Name == "" {
		return ProjectInput{}, validationError(ErrInvalidProjectInput, "name", "required", "name is required")
	}
	if !validHexColor(input.Color) {
		return ProjectInput{}, validationError(ErrInvalidProjectInput, "color", "invalid", "color must be a hex color")
	}
	if input.DefaultHourlyRateMinor != nil && *input.DefaultHourlyRateMinor < 0 {
		return ProjectInput{}, validationError(ErrInvalidProjectInput, "defaultHourlyRateMinor", "invalid", "defaultHourlyRateMinor must be non-negative")
	}
	if input.ClientID != "" {
		ok, err := s.activeClientExists(ctx, userID, input.ClientID)
		if err != nil {
			return ProjectInput{}, err
		}
		if !ok {
			return ProjectInput{}, validationError(ErrInvalidProjectInput, "clientId", "invalid", "clientId must reference an active client")
		}
	}

	return input, nil
}

func (s *Store) activeClientExists(ctx context.Context, userID string, clientID string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM clients
		WHERE user_id = ? AND id = ? AND archived_at IS NULL
	`, userID, clientID).Scan(&count); err != nil {
		return false, fmt.Errorf("check active client: %w", err)
	}
	return count > 0, nil
}

func validHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, char := range value[1:] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

func nullableInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}
