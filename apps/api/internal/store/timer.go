package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrTimerNotFound = errors.New("timer not found")
var ErrInvalidTimerInput = errors.New("invalid timer input")

type TimerStartInput struct {
	ClientID    string   `json:"clientId"`
	ProjectID   string   `json:"projectId"`
	TaskID      string   `json:"taskId"`
	TagIDs      []string `json:"tagIds"`
	Description string   `json:"description"`
	StartedAt   string   `json:"startedAt,omitempty"`
	Billable    bool     `json:"billable"`
}

func (s *Store) ListOpenTimers(ctx context.Context, userID string) ([]TimeEntry, error) {
	rows, err := s.db.QueryContext(ctx, timeEntrySelectSQL+`
		WHERE te.user_id = ? AND te.ended_at IS NULL AND te.source = 'timer'
		ORDER BY te.started_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list open timers: %w", err)
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
		return nil, fmt.Errorf("iterate open timers: %w", err)
	}

	if err := s.attachTimeEntryTags(ctx, entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func (s *Store) StartTimer(ctx context.Context, userID string, input TimerStartInput) (*TimeEntry, error) {
	normalized, err := s.normalizeTimerStartInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}

	billable, err := s.resolveBillableFlag(ctx, userID, normalized.ClientID, normalized.ProjectID, normalized.Billable, true)
	if err != nil {
		return nil, err
	}
	normalized.Billable = billable

	startedAt := truncateToMinute(time.Now().UTC())
	overlapWarning, err := s.hasTimeOverlap(ctx, userID, "", startedAt, startedAt.Add(time.Minute))
	if err != nil {
		return nil, err
	}

	timeEntryID, err := newID("ten")
	if err != nil {
		return nil, err
	}

	now := nowString()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin start timer: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO time_entries (
			id, user_id, client_id, project_id, task_id, description, started_at, ended_at,
			duration_seconds, billable, overlap_warning, source, sync_state, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL, 0, ?, ?, 'timer', 'synced', ?, ?)
	`, timeEntryID, userID, nullValue(normalized.ClientID), nullValue(normalized.ProjectID), nullValue(normalized.TaskID),
		normalized.Description, formatTime(startedAt), boolToInt(normalized.Billable), boolToInt(overlapWarning), now, now); err != nil {
		return nil, fmt.Errorf("insert timer: %w", err)
	}

	if err := s.replaceTimeEntryTags(ctx, tx, timeEntryID, normalized.TagIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit start timer: %w", err)
	}

	return s.TimeEntryByID(ctx, userID, timeEntryID)
}

func (s *Store) UpdateOpenTimer(ctx context.Context, userID string, timeEntryID string, input TimerStartInput) (*TimeEntry, error) {
	normalized, err := s.normalizeTimerStartInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}

	var startedAtUpdate *time.Time
	var overlapWarning *bool
	if strings.TrimSpace(input.StartedAt) != "" {
		startedAt, err := parseRFC3339(input.StartedAt)
		if err != nil {
			return nil, validationError(ErrInvalidTimeEntryInput, "startedAt", "invalid", "startedAt must be RFC3339")
		}
		startedAt = truncateToMinute(startedAt)
		now := truncateToMinute(time.Now().UTC())
		if startedAt.After(now) {
			return nil, validationError(ErrInvalidTimeEntryInput, "startedAt", "invalid", "startedAt cannot be in the future")
		}
		startedAtUpdate = &startedAt

		endedAt := truncateToMinute(time.Now().UTC())
		warning, err := s.hasTimeOverlap(ctx, userID, timeEntryID, *startedAtUpdate, endedAt)
		if err != nil {
			return nil, err
		}
		overlapWarning = &warning
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update timer: %w", err)
	}
	defer tx.Rollback()

	nowString := nowString()
	if startedAtUpdate != nil {
		result, err := tx.ExecContext(ctx, `
			UPDATE time_entries
			SET client_id = ?, project_id = ?, task_id = ?, description = ?, billable = ?, started_at = ?, overlap_warning = ?, updated_at = ?
			WHERE user_id = ? AND id = ? AND ended_at IS NULL AND source = 'timer'
		`, nullValue(normalized.ClientID), nullValue(normalized.ProjectID), nullValue(normalized.TaskID),
			normalized.Description, boolToInt(normalized.Billable), formatTime(*startedAtUpdate), boolToInt(*overlapWarning),
			nowString, userID, timeEntryID)
		if err != nil {
			return nil, fmt.Errorf("update timer: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("inspect update timer result: %w", err)
		}
		if affected == 0 {
			return nil, ErrTimerNotFound
		}
	} else {
		result, err := tx.ExecContext(ctx, `
			UPDATE time_entries
			SET client_id = ?, project_id = ?, task_id = ?, description = ?, billable = ?, updated_at = ?
			WHERE user_id = ? AND id = ? AND ended_at IS NULL AND source = 'timer'
		`, nullValue(normalized.ClientID), nullValue(normalized.ProjectID), nullValue(normalized.TaskID),
			normalized.Description, boolToInt(normalized.Billable), nowString, userID, timeEntryID)
		if err != nil {
			return nil, fmt.Errorf("update timer: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("inspect update timer result: %w", err)
		}
		if affected == 0 {
			return nil, ErrTimerNotFound
		}
	}

	if err := s.replaceTimeEntryTags(ctx, tx, timeEntryID, normalized.TagIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update timer: %w", err)
	}

	return s.TimeEntryByID(ctx, userID, timeEntryID)
}

func (s *Store) StopTimer(ctx context.Context, userID string, timeEntryID string) (*TimeEntry, error) {
	entry, err := s.openTimerByID(ctx, userID, timeEntryID)
	if err != nil {
		return nil, err
	}

	startedAt, err := parseRFC3339(entry.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("parse timer started_at: %w", err)
	}

	endedAt := truncateToMinute(time.Now().UTC())
	if !endedAt.After(startedAt) {
		endedAt = startedAt.Add(time.Minute)
	}

	overlapWarning, err := s.hasTimeOverlap(ctx, userID, timeEntryID, startedAt, endedAt)
	if err != nil {
		return nil, err
	}

	durationSeconds := int(endedAt.Sub(startedAt).Seconds())

	result, err := s.db.ExecContext(ctx, `
		UPDATE time_entries
		SET ended_at = ?, duration_seconds = ?, overlap_warning = ?, updated_at = ?
		WHERE user_id = ? AND id = ? AND ended_at IS NULL AND source = 'timer'
	`, formatTime(endedAt), durationSeconds, boolToInt(overlapWarning), nowString(), userID, timeEntryID)
	if err != nil {
		return nil, fmt.Errorf("stop timer: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect stop timer result: %w", err)
	}
	if affected == 0 {
		return nil, ErrTimerNotFound
	}

	return s.TimeEntryByID(ctx, userID, timeEntryID)
}

func (s *Store) DiscardTimer(ctx context.Context, userID string, timeEntryID string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM time_entries
		WHERE user_id = ? AND id = ? AND ended_at IS NULL AND source = 'timer'
	`, userID, timeEntryID)
	if err != nil {
		return fmt.Errorf("discard timer: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect discard timer result: %w", err)
	}
	if affected == 0 {
		return ErrTimerNotFound
	}
	return nil
}

func (s *Store) openTimerByID(ctx context.Context, userID string, timeEntryID string) (*TimeEntry, error) {
	entry, err := queryTimeEntry(ctx, s.db, timeEntrySelectSQL+`
		WHERE te.user_id = ? AND te.id = ? AND te.ended_at IS NULL AND te.source = 'timer'
	`, userID, timeEntryID)
	if err != nil {
		if errors.Is(err, ErrTimeEntryNotFound) {
			return nil, ErrTimerNotFound
		}
		return nil, err
	}
	return entry, nil
}

func (s *Store) normalizeTimerStartInput(ctx context.Context, userID string, input TimerStartInput) (TimerStartInput, error) {
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.TaskID = strings.TrimSpace(input.TaskID)
	input.Description = strings.TrimSpace(input.Description)

	relations, err := s.normalizeTimeEntryRelations(ctx, userID, TimeEntryInput{
		ClientID:  input.ClientID,
		ProjectID: input.ProjectID,
		TaskID:    input.TaskID,
		TagIDs:    input.TagIDs,
	})
	if err != nil {
		return TimerStartInput{}, err
	}

	return TimerStartInput{
		ClientID:    relations.ClientID,
		ProjectID:   relations.ProjectID,
		TaskID:      relations.TaskID,
		TagIDs:      relations.TagIDs,
		Description: input.Description,
		Billable:    input.Billable,
	}, nil
}

func (s *Store) normalizeTimeEntryRelations(ctx context.Context, userID string, input TimeEntryInput) (TimeEntryInput, error) {
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.TaskID = strings.TrimSpace(input.TaskID)

	if input.ClientID != "" {
		ok, err := s.activeClientExists(ctx, userID, input.ClientID)
		if err != nil {
			return TimeEntryInput{}, err
		}
		if !ok {
			return TimeEntryInput{}, validationError(ErrInvalidTimeEntryInput, "clientId", "invalid", "clientId must reference an active client")
		}
	}

	if input.TaskID != "" {
		task, err := s.TaskByID(ctx, userID, input.TaskID)
		if err != nil {
			if errors.Is(err, ErrTaskNotFound) {
				return TimeEntryInput{}, validationError(ErrInvalidTimeEntryInput, "taskId", "invalid", "taskId must reference an active task")
			}
			return TimeEntryInput{}, err
		}
		if task.ArchivedAt != "" {
			return TimeEntryInput{}, validationError(ErrInvalidTimeEntryInput, "taskId", "invalid", "taskId must reference an active task")
		}
		if task.ProjectID != "" {
			if input.ProjectID == "" {
				input.ProjectID = task.ProjectID
			} else if input.ProjectID != task.ProjectID {
				return TimeEntryInput{}, validationError(ErrInvalidTimeEntryInput, "projectId", "invalid", "projectId must match the selected task project")
			}
		}
	}

	if input.ProjectID != "" {
		project, err := s.ProjectByID(ctx, userID, input.ProjectID)
		if err != nil {
			if errors.Is(err, ErrProjectNotFound) {
				return TimeEntryInput{}, validationError(ErrInvalidTimeEntryInput, "projectId", "invalid", "projectId must reference an active project")
			}
			return TimeEntryInput{}, err
		}
		if project.ArchivedAt != "" {
			return TimeEntryInput{}, validationError(ErrInvalidTimeEntryInput, "projectId", "invalid", "projectId must reference an active project")
		}
		if project.ClientID != "" {
			input.ClientID = project.ClientID
		}
	}

	if err := s.validateTagIDs(ctx, userID, input.TagIDs); err != nil {
		return TimeEntryInput{}, err
	}

	return input, nil
}
