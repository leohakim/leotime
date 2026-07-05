package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrTaskNotFound = errors.New("task not found")
var ErrInvalidTaskInput = errors.New("invalid task input")

type Task struct {
	ID           string `json:"id"`
	ProjectID    string `json:"projectId"`
	ProjectName  string `json:"projectName"`
	ProjectColor string `json:"projectColor"`
	Name         string `json:"name"`
	Billable     bool   `json:"billable"`
	ArchivedAt   string `json:"archivedAt"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type TaskInput struct {
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Billable  bool   `json:"billable"`
}

func (s *Store) ListTasks(ctx context.Context, userID string, includeArchived bool, projectID string) ([]Task, error) {
	query := `
		SELECT t.id, t.project_id, COALESCE(p.name, ''), COALESCE(p.color, ''), t.name, t.billable,
			t.archived_at, t.created_at, t.updated_at
		FROM tasks t
		LEFT JOIN projects p ON p.id = t.project_id AND p.user_id = t.user_id
		WHERE t.user_id = ?
	`
	args := []any{userID}
	if !includeArchived {
		query += " AND t.archived_at IS NULL"
	}
	if strings.TrimSpace(projectID) != "" {
		query += " AND t.project_id = ?"
		args = append(args, strings.TrimSpace(projectID))
	}
	query += " ORDER BY t.created_at DESC, t.id DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (s *Store) TaskByID(ctx context.Context, userID string, taskID string) (*Task, error) {
	task, err := queryTask(ctx, s.db, `
		SELECT t.id, t.project_id, COALESCE(p.name, ''), COALESCE(p.color, ''), t.name, t.billable,
			t.archived_at, t.created_at, t.updated_at
		FROM tasks t
		LEFT JOIN projects p ON p.id = t.project_id AND p.user_id = t.user_id
		WHERE t.user_id = ? AND t.id = ?
	`, userID, taskID)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Store) CreateTask(ctx context.Context, userID string, input TaskInput) (*Task, error) {
	normalized, err := s.normalizeTaskInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}

	taskID, err := newID("tsk")
	if err != nil {
		return nil, err
	}
	now := nowString()

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, user_id, project_id, name, billable, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, taskID, userID, nullValue(normalized.ProjectID), normalized.Name, boolToInt(normalized.Billable), now, now); err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	return s.TaskByID(ctx, userID, taskID)
}

func (s *Store) UpdateTask(ctx context.Context, userID string, taskID string, input TaskInput) (*Task, error) {
	normalized, err := s.normalizeTaskInput(ctx, userID, input)
	if err != nil {
		return nil, err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET project_id = ?, name = ?, billable = ?, updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nullValue(normalized.ProjectID), normalized.Name, boolToInt(normalized.Billable), nowString(), userID, taskID)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect update task result: %w", err)
	}
	if affected == 0 {
		return nil, ErrTaskNotFound
	}

	return s.TaskByID(ctx, userID, taskID)
}

func (s *Store) ArchiveTask(ctx context.Context, userID string, taskID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET archived_at = COALESCE(archived_at, ?), updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nowString(), nowString(), userID, taskID)
	if err != nil {
		return fmt.Errorf("archive task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect archive task result: %w", err)
	}
	if affected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func queryTask(ctx context.Context, db *sql.DB, query string, args ...any) (*Task, error) {
	task, err := scanTask(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return &task, nil
}

func scanTask(scanner taskScanner) (Task, error) {
	var task Task
	var projectID sql.NullString
	var billable int
	var archivedAt sql.NullString

	if err := scanner.Scan(
		&task.ID,
		&projectID,
		&task.ProjectName,
		&task.ProjectColor,
		&task.Name,
		&billable,
		&archivedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return Task{}, fmt.Errorf("scan task: %w", err)
	}

	task.ProjectID = projectID.String
	task.Billable = billable != 0
	task.ArchivedAt = archivedAt.String
	return task, nil
}

func (s *Store) normalizeTaskInput(ctx context.Context, userID string, input TaskInput) (TaskInput, error) {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Name = strings.TrimSpace(input.Name)

	if input.Name == "" {
		return TaskInput{}, fmt.Errorf("%w: name is required", ErrInvalidTaskInput)
	}

	required, err := s.taskProjectRequired(ctx, userID)
	if err != nil {
		return TaskInput{}, err
	}
	if required && input.ProjectID == "" {
		return TaskInput{}, fmt.Errorf("%w: projectId is required by user settings", ErrInvalidTaskInput)
	}
	if input.ProjectID != "" {
		ok, err := s.activeProjectExists(ctx, userID, input.ProjectID)
		if err != nil {
			return TaskInput{}, err
		}
		if !ok {
			return TaskInput{}, fmt.Errorf("%w: projectId must reference an active project", ErrInvalidTaskInput)
		}
	}

	return input, nil
}

func (s *Store) taskProjectRequired(ctx context.Context, userID string) (bool, error) {
	var required int
	if err := s.db.QueryRowContext(ctx, `
		SELECT task_project_required
		FROM app_settings
		WHERE user_id = ?
	`, userID).Scan(&required); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("read task project setting: %w", err)
	}
	return required != 0, nil
}

func (s *Store) activeProjectExists(ctx context.Context, userID string, projectID string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM projects
		WHERE user_id = ? AND id = ? AND archived_at IS NULL
	`, userID, projectID).Scan(&count); err != nil {
		return false, fmt.Errorf("check active project: %w", err)
	}
	return count > 0, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
