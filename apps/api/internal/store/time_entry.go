package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrTimeEntryNotFound = errors.New("time entry not found")
var ErrInvalidTimeEntryInput = errors.New("invalid time entry input")

type TimeEntryTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TimeEntry struct {
	ID              string         `json:"id"`
	ClientID        string         `json:"clientId"`
	ClientName      string         `json:"clientName"`
	ProjectID       string         `json:"projectId"`
	ProjectName     string         `json:"projectName"`
	ProjectColor    string         `json:"projectColor"`
	TaskID          string         `json:"taskId"`
	TaskName        string         `json:"taskName"`
	Description     string         `json:"description"`
	StartedAt       string         `json:"startedAt"`
	EndedAt         string         `json:"endedAt"`
	DurationSeconds int            `json:"durationSeconds"`
	Billable        bool           `json:"billable"`
	OverlapWarning  bool           `json:"overlapWarning"`
	Source          string         `json:"source"`
	Tags            []TimeEntryTag `json:"tags"`
	CreatedAt       string         `json:"createdAt"`
	UpdatedAt       string         `json:"updatedAt"`
}

type TimeEntryInput struct {
	ClientID    string   `json:"clientId"`
	ProjectID   string   `json:"projectId"`
	TaskID      string   `json:"taskId"`
	TagIDs      []string `json:"tagIds"`
	Description string   `json:"description"`
	StartedAt   string   `json:"startedAt"`
	EndedAt     string   `json:"endedAt"`
	Billable    bool     `json:"billable"`
}

type TimeEntryListOptions struct {
	From      string
	To        string
	ClientID  string
	ProjectID string
	TaskID    string
}

func (s *Store) ListTimeEntries(ctx context.Context, userID string, options TimeEntryListOptions) ([]TimeEntry, error) {
	query := `
		SELECT te.id, te.client_id, COALESCE(c.name, ''), te.project_id, COALESCE(p.name, ''), COALESCE(p.color, ''),
			te.task_id, COALESCE(t.name, ''), te.description, te.started_at, te.ended_at, te.duration_seconds,
			te.billable, te.overlap_warning, te.source, te.created_at, te.updated_at
		FROM time_entries te
		LEFT JOIN clients c ON c.id = te.client_id AND c.user_id = te.user_id
		LEFT JOIN projects p ON p.id = te.project_id AND p.user_id = te.user_id
		LEFT JOIN tasks t ON t.id = te.task_id AND t.user_id = te.user_id
		WHERE te.user_id = ? AND te.ended_at IS NOT NULL
	`
	args := []any{userID}

	if strings.TrimSpace(options.From) != "" {
		query += " AND te.started_at >= ?"
		args = append(args, strings.TrimSpace(options.From))
	}
	if strings.TrimSpace(options.To) != "" {
		query += " AND te.started_at <= ?"
		args = append(args, strings.TrimSpace(options.To))
	}
	if strings.TrimSpace(options.ClientID) != "" {
		query += " AND te.client_id = ?"
		args = append(args, strings.TrimSpace(options.ClientID))
	}
	if strings.TrimSpace(options.ProjectID) != "" {
		query += " AND te.project_id = ?"
		args = append(args, strings.TrimSpace(options.ProjectID))
	}
	if strings.TrimSpace(options.TaskID) != "" {
		query += " AND te.task_id = ?"
		args = append(args, strings.TrimSpace(options.TaskID))
	}

	query += " ORDER BY te.started_at DESC LIMIT 500"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list time entries: %w", err)
	}
	defer rows.Close()

	var entries []TimeEntry
	for rows.Next() {
		entry, err := scanTimeEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate time entries: %w", err)
	}

	if err := s.attachTimeEntryTags(ctx, entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func (s *Store) TimeEntryByID(ctx context.Context, userID string, timeEntryID string) (*TimeEntry, error) {
	entry, err := queryTimeEntry(ctx, s.db, `
		SELECT te.id, te.client_id, COALESCE(c.name, ''), te.project_id, COALESCE(p.name, ''), COALESCE(p.color, ''),
			te.task_id, COALESCE(t.name, ''), te.description, te.started_at, te.ended_at, te.duration_seconds,
			te.billable, te.overlap_warning, te.source, te.created_at, te.updated_at
		FROM time_entries te
		LEFT JOIN clients c ON c.id = te.client_id AND c.user_id = te.user_id
		LEFT JOIN projects p ON p.id = te.project_id AND p.user_id = te.user_id
		LEFT JOIN tasks t ON t.id = te.task_id AND t.user_id = te.user_id
		WHERE te.user_id = ? AND te.id = ?
	`, userID, timeEntryID)
	if err != nil {
		return nil, err
	}

	entries := []TimeEntry{*entry}
	if err := s.attachTimeEntryTags(ctx, entries); err != nil {
		return nil, err
	}

	return &entries[0], nil
}

func (s *Store) CreateTimeEntry(ctx context.Context, userID string, input TimeEntryInput) (*TimeEntry, error) {
	normalized, startedAt, endedAt, err := s.normalizeTimeEntryInput(ctx, userID, "", input)
	if err != nil {
		return nil, err
	}

	timeEntryID, err := newID("ten")
	if err != nil {
		return nil, err
	}

	overlapWarning, err := s.hasTimeOverlap(ctx, userID, "", startedAt, endedAt)
	if err != nil {
		return nil, err
	}

	durationSeconds := int(endedAt.Sub(startedAt).Seconds())
	now := nowString()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create time entry: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO time_entries (
			id, user_id, client_id, project_id, task_id, description, started_at, ended_at,
			duration_seconds, billable, overlap_warning, source, sync_state, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'manual', 'synced', ?, ?)
	`, timeEntryID, userID, nullValue(normalized.ClientID), nullValue(normalized.ProjectID), nullValue(normalized.TaskID),
		normalized.Description, formatTime(startedAt), formatTime(endedAt), durationSeconds, boolToInt(normalized.Billable),
		boolToInt(overlapWarning), now, now); err != nil {
		return nil, fmt.Errorf("insert time entry: %w", err)
	}

	if err := s.replaceTimeEntryTags(ctx, tx, timeEntryID, normalized.TagIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create time entry: %w", err)
	}

	return s.TimeEntryByID(ctx, userID, timeEntryID)
}

func (s *Store) UpdateTimeEntry(ctx context.Context, userID string, timeEntryID string, input TimeEntryInput) (*TimeEntry, error) {
	normalized, startedAt, endedAt, err := s.normalizeTimeEntryInput(ctx, userID, timeEntryID, input)
	if err != nil {
		return nil, err
	}

	overlapWarning, err := s.hasTimeOverlap(ctx, userID, timeEntryID, startedAt, endedAt)
	if err != nil {
		return nil, err
	}

	durationSeconds := int(endedAt.Sub(startedAt).Seconds())

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update time entry: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE time_entries
		SET client_id = ?, project_id = ?, task_id = ?, description = ?, started_at = ?, ended_at = ?,
			duration_seconds = ?, billable = ?, overlap_warning = ?, source = 'manual', updated_at = ?
		WHERE user_id = ? AND id = ? AND ended_at IS NOT NULL
	`, nullValue(normalized.ClientID), nullValue(normalized.ProjectID), nullValue(normalized.TaskID), normalized.Description,
		formatTime(startedAt), formatTime(endedAt), durationSeconds, boolToInt(normalized.Billable), boolToInt(overlapWarning),
		nowString(), userID, timeEntryID)
	if err != nil {
		return nil, fmt.Errorf("update time entry: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect update time entry result: %w", err)
	}
	if affected == 0 {
		return nil, ErrTimeEntryNotFound
	}

	if err := s.replaceTimeEntryTags(ctx, tx, timeEntryID, normalized.TagIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update time entry: %w", err)
	}

	return s.TimeEntryByID(ctx, userID, timeEntryID)
}

func (s *Store) DeleteTimeEntry(ctx context.Context, userID string, timeEntryID string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM time_entries
		WHERE user_id = ? AND id = ? AND ended_at IS NOT NULL
	`, userID, timeEntryID)
	if err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect delete time entry result: %w", err)
	}
	if affected == 0 {
		return ErrTimeEntryNotFound
	}
	return nil
}

type timeEntryScanner interface {
	Scan(dest ...any) error
}

func queryTimeEntry(ctx context.Context, db *sql.DB, query string, args ...any) (*TimeEntry, error) {
	entry, err := scanTimeEntry(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTimeEntryNotFound
		}
		return nil, err
	}
	return &entry, nil
}

func scanTimeEntry(scanner timeEntryScanner) (TimeEntry, error) {
	var entry TimeEntry
	var clientID sql.NullString
	var projectID sql.NullString
	var taskID sql.NullString
	var billable int
	var overlapWarning int

	if err := scanner.Scan(
		&entry.ID,
		&clientID,
		&entry.ClientName,
		&projectID,
		&entry.ProjectName,
		&entry.ProjectColor,
		&taskID,
		&entry.TaskName,
		&entry.Description,
		&entry.StartedAt,
		&entry.EndedAt,
		&entry.DurationSeconds,
		&billable,
		&overlapWarning,
		&entry.Source,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return TimeEntry{}, fmt.Errorf("scan time entry: %w", err)
	}

	entry.ClientID = clientID.String
	entry.ProjectID = projectID.String
	entry.TaskID = taskID.String
	entry.Billable = billable != 0
	entry.OverlapWarning = overlapWarning != 0
	entry.Tags = []TimeEntryTag{}
	return entry, nil
}

func (s *Store) normalizeTimeEntryInput(ctx context.Context, userID string, timeEntryID string, input TimeEntryInput) (TimeEntryInput, time.Time, time.Time, error) {
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.TaskID = strings.TrimSpace(input.TaskID)
	input.Description = strings.TrimSpace(input.Description)
	input.StartedAt = strings.TrimSpace(input.StartedAt)
	input.EndedAt = strings.TrimSpace(input.EndedAt)

	if input.StartedAt == "" || input.EndedAt == "" {
		return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: startedAt and endedAt are required", ErrInvalidTimeEntryInput)
	}

	startedAt, err := parseRFC3339(input.StartedAt)
	if err != nil {
		return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: startedAt must be RFC3339", ErrInvalidTimeEntryInput)
	}
	endedAt, err := parseRFC3339(input.EndedAt)
	if err != nil {
		return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: endedAt must be RFC3339", ErrInvalidTimeEntryInput)
	}

	startedAt = truncateToMinute(startedAt)
	endedAt = truncateToMinute(endedAt)
	if !endedAt.After(startedAt) {
		return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: endedAt must be after startedAt", ErrInvalidTimeEntryInput)
	}
	if endedAt.Sub(startedAt) < time.Minute {
		return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: duration must be at least one minute", ErrInvalidTimeEntryInput)
	}

	if input.ClientID != "" {
		ok, err := s.activeClientExists(ctx, userID, input.ClientID)
		if err != nil {
			return TimeEntryInput{}, time.Time{}, time.Time{}, err
		}
		if !ok {
			return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: clientId must reference an active client", ErrInvalidTimeEntryInput)
		}
	}

	if input.TaskID != "" {
		task, err := s.TaskByID(ctx, userID, input.TaskID)
		if err != nil {
			if errors.Is(err, ErrTaskNotFound) {
				return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: taskId must reference an active task", ErrInvalidTimeEntryInput)
			}
			return TimeEntryInput{}, time.Time{}, time.Time{}, err
		}
		if task.ArchivedAt != "" {
			return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: taskId must reference an active task", ErrInvalidTimeEntryInput)
		}
		if task.ProjectID != "" {
			if input.ProjectID == "" {
				input.ProjectID = task.ProjectID
			} else if input.ProjectID != task.ProjectID {
				return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: projectId must match the selected task project", ErrInvalidTimeEntryInput)
			}
		}
	}

	if input.ProjectID != "" {
		project, err := s.ProjectByID(ctx, userID, input.ProjectID)
		if err != nil {
			if errors.Is(err, ErrProjectNotFound) {
				return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: projectId must reference an active project", ErrInvalidTimeEntryInput)
			}
			return TimeEntryInput{}, time.Time{}, time.Time{}, err
		}
		if project.ArchivedAt != "" {
			return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: projectId must reference an active project", ErrInvalidTimeEntryInput)
		}
		if project.ClientID != "" {
			if input.ClientID == "" {
				input.ClientID = project.ClientID
			} else if input.ClientID != project.ClientID {
				return TimeEntryInput{}, time.Time{}, time.Time{}, fmt.Errorf("%w: clientId must match the selected project client", ErrInvalidTimeEntryInput)
			}
		}
	}

	if err := s.validateTagIDs(ctx, userID, input.TagIDs); err != nil {
		return TimeEntryInput{}, time.Time{}, time.Time{}, err
	}

	return input, startedAt, endedAt, nil
}

func (s *Store) validateTagIDs(ctx context.Context, userID string, tagIDs []string) error {
	for _, tagID := range tagIDs {
		tagID = strings.TrimSpace(tagID)
		if tagID == "" {
			continue
		}
		if _, err := s.TagByID(ctx, userID, tagID); err != nil {
			if errors.Is(err, ErrTagNotFound) {
				return fmt.Errorf("%w: tagIds must reference existing tags", ErrInvalidTimeEntryInput)
			}
			return err
		}
	}
	return nil
}

func (s *Store) hasTimeOverlap(ctx context.Context, userID string, entryID string, startedAt time.Time, endedAt time.Time) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM time_entries
		WHERE user_id = ?
			AND id != ?
			AND started_at < ?
			AND COALESCE(ended_at, '9999-12-31T23:59:59Z') > ?
	`, userID, entryID, formatTime(endedAt), formatTime(startedAt)).Scan(&count); err != nil {
		return false, fmt.Errorf("check time entry overlap: %w", err)
	}
	return count > 0, nil
}

func (s *Store) attachTimeEntryTags(ctx context.Context, entries []TimeEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ids := make([]string, 0, len(entries))
	index := make(map[string]int, len(entries))
	for i, entry := range entries {
		ids = append(ids, entry.ID)
		index[entry.ID] = i
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(`
		SELECT tet.time_entry_id, t.id, t.name, t.color
		FROM time_entry_tags tet
		JOIN tags t ON t.id = tet.tag_id
		WHERE tet.time_entry_id IN (%s)
		ORDER BY lower(t.name)
	`, placeholders)

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("load time entry tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var timeEntryID string
		var tag TimeEntryTag
		if err := rows.Scan(&timeEntryID, &tag.ID, &tag.Name, &tag.Color); err != nil {
			return fmt.Errorf("scan time entry tag: %w", err)
		}
		entryIndex, ok := index[timeEntryID]
		if !ok {
			continue
		}
		entries[entryIndex].Tags = append(entries[entryIndex].Tags, tag)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate time entry tags: %w", err)
	}

	return nil
}

func (s *Store) replaceTimeEntryTags(ctx context.Context, tx *sql.Tx, timeEntryID string, tagIDs []string) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM time_entry_tags WHERE time_entry_id = ?", timeEntryID); err != nil {
		return fmt.Errorf("clear time entry tags: %w", err)
	}

	for _, tagID := range tagIDs {
		tagID = strings.TrimSpace(tagID)
		if tagID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO time_entry_tags (time_entry_id, tag_id)
			VALUES (?, ?)
		`, timeEntryID, tagID); err != nil {
			return fmt.Errorf("insert time entry tag: %w", err)
		}
	}

	return nil
}

func truncateToMinute(value time.Time) time.Time {
	return value.UTC().Truncate(time.Minute)
}

func parseRFC3339(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Parse(time.RFC3339, value)
	}
	return parsed, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
