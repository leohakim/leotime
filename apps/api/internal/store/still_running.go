package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type StillRunningCandidate struct {
	TimeEntryID    string
	UserID         string
	UserEmail      string
	UserName       string
	Locale         string
	ProjectName    string
	TaskName       string
	Description    string
	StartedAt      string
	ThresholdHours int
}

func (s *Store) ListStillRunningNotificationCandidates(ctx context.Context, now time.Time, limit int) ([]StillRunningCandidate, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT te.id, te.project_id, COALESCE(p.name, ''), te.task_id, COALESCE(t.name, ''),
			te.description, te.started_at, u.id, u.email, u.name, u.locale,
			COALESCE(a.timer_still_running_hours, 8)
		FROM time_entries te
		JOIN users u ON u.id = te.user_id
		LEFT JOIN app_settings a ON a.user_id = te.user_id
		LEFT JOIN projects p ON p.id = te.project_id AND p.user_id = te.user_id
		LEFT JOIN tasks t ON t.id = te.task_id AND t.user_id = te.user_id
		WHERE te.ended_at IS NULL
			AND te.source = 'timer'
			AND (te.still_active_email_sent_at IS NULL OR te.still_active_email_sent_at = '')
			AND COALESCE(a.timer_still_running_enabled, 1) = 1
			AND NOT EXISTS (
				SELECT 1
				FROM email_outbox o
				WHERE o.time_entry_id = te.id
					AND o.kind = 'timer_still_running'
					AND o.status IN ('pending', 'sent')
			)
		ORDER BY te.started_at ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list still running notification candidates: %w", err)
	}
	defer rows.Close()

	var candidates []StillRunningCandidate
	for rows.Next() {
		var candidate StillRunningCandidate
		var projectID sql.NullString
		var taskID sql.NullString
		if err := rows.Scan(
			&candidate.TimeEntryID,
			&projectID,
			&candidate.ProjectName,
			&taskID,
			&candidate.TaskName,
			&candidate.Description,
			&candidate.StartedAt,
			&candidate.UserID,
			&candidate.UserEmail,
			&candidate.UserName,
			&candidate.Locale,
			&candidate.ThresholdHours,
		); err != nil {
			return nil, fmt.Errorf("scan still running candidate: %w", err)
		}

		startedAt, err := parseRFC3339(candidate.StartedAt)
		if err != nil {
			return nil, fmt.Errorf("parse started_at for %s: %w", candidate.TimeEntryID, err)
		}
		if candidate.ThresholdHours <= 0 {
			candidate.ThresholdHours = 8
		}
		threshold := time.Duration(candidate.ThresholdHours) * time.Hour
		if now.Sub(startedAt) < threshold {
			continue
		}

		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate still running candidates: %w", err)
	}

	return candidates, nil
}

func (s *Store) MarkStillRunningEmailSent(ctx context.Context, timeEntryID string, sentAt time.Time) error {
	timeEntryID = strings.TrimSpace(timeEntryID)
	if timeEntryID == "" {
		return fmt.Errorf("time entry id is required")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE time_entries
		SET still_active_email_sent_at = ?, updated_at = ?
		WHERE id = ? AND ended_at IS NULL
	`, formatTime(sentAt), formatTime(sentAt), timeEntryID)
	if err != nil {
		return fmt.Errorf("mark still running email sent: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect still running email sent result: %w", err)
	}
	if affected == 0 {
		return ErrTimeEntryNotFound
	}
	return nil
}
