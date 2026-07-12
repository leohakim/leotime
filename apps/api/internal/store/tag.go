package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrTagNotFound = errors.New("tag not found")
var ErrInvalidTagInput = errors.New("invalid tag input")
var ErrDuplicateTagName = errors.New("duplicate tag name")

type Tag struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	ArchivedAt string `json:"archivedAt"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type TagInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TagSummary struct {
	Active   int `json:"active"`
	Archived int `json:"archived"`
}

func (s *Store) TagSummary(ctx context.Context, userID string) (*TagSummary, error) {
	const query = `
		SELECT
			COALESCE(SUM(CASE WHEN archived_at IS NULL THEN 1 ELSE 0 END), 0) AS active_count,
			COALESCE(SUM(CASE WHEN archived_at IS NOT NULL THEN 1 ELSE 0 END), 0) AS archived_count
		FROM tags
		WHERE user_id = ?
	`
	var active, archived int
	if err := s.db.QueryRowContext(ctx, query, userID).Scan(&active, &archived); err != nil {
		return nil, fmt.Errorf("tag summary: %w", err)
	}
	return &TagSummary{Active: active, Archived: archived}, nil
}

func (s *Store) ListTags(ctx context.Context, userID string, includeArchived bool) ([]Tag, error) {
	query := `
		SELECT id, name, color, archived_at, created_at, updated_at
		FROM tags
		WHERE user_id = ?
	`
	if !includeArchived {
		query += " AND archived_at IS NULL"
	}
	query += " ORDER BY lower(name), created_at"

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		tag, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return tags, nil
}

func (s *Store) TagByID(ctx context.Context, userID string, tagID string) (*Tag, error) {
	tag, err := queryTag(ctx, s.db, `
		SELECT id, name, color, archived_at, created_at, updated_at
		FROM tags
		WHERE user_id = ? AND id = ?
	`, userID, tagID)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *Store) CreateTag(ctx context.Context, userID string, input TagInput) (*Tag, error) {
	normalized, err := s.normalizeTagInput(ctx, userID, "", input)
	if err != nil {
		return nil, err
	}

	tagID, err := newID("tag")
	if err != nil {
		return nil, err
	}
	now := nowString()

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO tags (id, user_id, name, color, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tagID, userID, normalized.Name, normalized.Color, now, now); err != nil {
		if isUniqueConstraintError(err) {
			return nil, validationError(ErrDuplicateTagName, "name", "duplicate", "name must be unique")
		}
		return nil, fmt.Errorf("insert tag: %w", err)
	}

	return s.TagByID(ctx, userID, tagID)
}

func (s *Store) UpdateTag(ctx context.Context, userID string, tagID string, input TagInput) (*Tag, error) {
	normalized, err := s.normalizeTagInput(ctx, userID, tagID, input)
	if err != nil {
		return nil, err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE tags
		SET name = ?, color = ?, updated_at = ?
		WHERE user_id = ? AND id = ?
	`, normalized.Name, normalized.Color, nowString(), userID, tagID)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, validationError(ErrDuplicateTagName, "name", "duplicate", "name must be unique")
		}
		return nil, fmt.Errorf("update tag: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect update tag result: %w", err)
	}
	if affected == 0 {
		return nil, ErrTagNotFound
	}

	return s.TagByID(ctx, userID, tagID)
}

func (s *Store) ArchiveTag(ctx context.Context, userID string, tagID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tags
		SET archived_at = COALESCE(archived_at, ?), updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nowString(), nowString(), userID, tagID)
	if err != nil {
		return fmt.Errorf("archive tag: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect archive tag result: %w", err)
	}
	if affected == 0 {
		return ErrTagNotFound
	}
	return nil
}

func (s *Store) RestoreTag(ctx context.Context, userID string, tagID string) (*Tag, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tags
		SET archived_at = NULL, updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nowString(), userID, tagID)
	if err != nil {
		return nil, fmt.Errorf("restore tag: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect restore tag result: %w", err)
	}
	if affected == 0 {
		return nil, ErrTagNotFound
	}

	return s.TagByID(ctx, userID, tagID)
}

type tagScanner interface {
	Scan(dest ...any) error
}

func queryTag(ctx context.Context, db *sql.DB, query string, args ...any) (*Tag, error) {
	tag, err := scanTag(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	return &tag, nil
}

func scanTag(scanner tagScanner) (Tag, error) {
	var tag Tag
	var archivedAt sql.NullString
	if err := scanner.Scan(&tag.ID, &tag.Name, &tag.Color, &archivedAt, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
		return Tag{}, fmt.Errorf("scan tag: %w", err)
	}
	tag.ArchivedAt = archivedAt.String
	return tag, nil
}

func (s *Store) normalizeTagInput(ctx context.Context, userID string, tagID string, input TagInput) (TagInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Color = strings.TrimSpace(input.Color)
	if input.Color == "" {
		input.Color = "#64748b"
	}

	if input.Name == "" {
		return TagInput{}, validationError(ErrInvalidTagInput, "name", "required", "name is required")
	}
	if !validHexColor(input.Color) {
		return TagInput{}, validationError(ErrInvalidTagInput, "color", "invalid", "color must be a hex color")
	}

	exists, err := s.tagNameTaken(ctx, userID, input.Name, tagID)
	if err != nil {
		return TagInput{}, err
	}
	if exists {
		return TagInput{}, validationError(ErrDuplicateTagName, "name", "duplicate", "name must be unique")
	}

	return input, nil
}

func (s *Store) tagNameTaken(ctx context.Context, userID string, name string, excludeTagID string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM tags
		WHERE user_id = ? AND lower(name) = lower(?) AND archived_at IS NULL`
	args := []any{userID, name}
	if excludeTagID != "" {
		query += " AND id <> ?"
		args = append(args, excludeTagID)
	}

	var count int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("check tag name: %w", err)
	}
	return count > 0, nil
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}
