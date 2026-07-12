package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrDailySummaryNotFound = errors.New("daily summary not found")
var ErrDailySummaryApproved = errors.New("daily summary is approved")

type DailySummaryStatus string

const (
	DailySummaryDraft    DailySummaryStatus = "draft"
	DailySummaryApproved DailySummaryStatus = "approved"
)

type DailySummaryRecord struct {
	ID               string              `json:"id"`
	Date             string              `json:"date"`
	Status           DailySummaryStatus  `json:"status"`
	DraftText        string              `json:"draftText"`
	ApprovedText     string              `json:"approvedText"`
	ManualNote       string              `json:"manualNote"`
	Options          DailySummaryOptions `json:"options"`
	GenerationSource string              `json:"generationSource"`
	GenerationCount  int                 `json:"generationCount"`
	ContextJSON      string              `json:"contextJson,omitempty"`
	ApprovedAt       string              `json:"approvedAt"`
	CreatedAt        string              `json:"createdAt"`
	UpdatedAt        string              `json:"updatedAt"`
}

type DailySummaryRecordInput struct {
	DraftText        string
	ManualNote       string
	Options          DailySummaryOptions
	GenerationSource string
	ContextJSON      string
	IncrementCount   bool
}

func (s *Store) DailySummaryByDate(ctx context.Context, userID string, date string) (*DailySummaryRecord, error) {
	date = strings.TrimSpace(date)
	row := s.db.QueryRowContext(ctx, `
		SELECT id, summary_date, status, draft_text, approved_text, manual_note, options_json,
			generation_source, generation_count, context_json, COALESCE(approved_at, ''), created_at, updated_at
		FROM daily_summary_records
		WHERE user_id = ? AND summary_date = ?
	`, userID, date)

	record, err := scanDailySummaryRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDailySummaryNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (s *Store) UpsertDailySummaryDraft(ctx context.Context, userID string, date string, input DailySummaryRecordInput) (*DailySummaryRecord, error) {
	date = strings.TrimSpace(date)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, validationError(ErrInvalidTimeEntryInput, "date", "invalid", "date must be YYYY-MM-DD")
	}

	existing, err := s.DailySummaryByDate(ctx, userID, date)
	if err != nil && !errors.Is(err, ErrDailySummaryNotFound) {
		return nil, err
	}
	if existing != nil && existing.Status == DailySummaryApproved {
		return nil, ErrDailySummaryApproved
	}

	optionsJSON, err := json.Marshal(input.Options)
	if err != nil {
		return nil, fmt.Errorf("marshal summary options: %w", err)
	}
	contextJSON := strings.TrimSpace(input.ContextJSON)
	if contextJSON == "" {
		contextJSON = "{}"
	}
	generationSource := strings.TrimSpace(input.GenerationSource)
	if generationSource == "" {
		generationSource = "template"
	}
	now := nowString()

	if existing == nil {
		recordID, err := newID("dsm")
		if err != nil {
			return nil, err
		}
		generationCount := 0
		if input.IncrementCount {
			generationCount = 1
		}
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO daily_summary_records (
				id, user_id, summary_date, status, draft_text, approved_text, manual_note, options_json,
				generation_source, generation_count, context_json, created_at, updated_at
			) VALUES (?, ?, ?, 'draft', ?, '', ?, ?, ?, ?, ?, ?, ?)
		`, recordID, userID, date, strings.TrimSpace(input.DraftText), strings.TrimSpace(input.ManualNote),
			string(optionsJSON), generationSource, generationCount, contextJSON, now, now); err != nil {
			return nil, fmt.Errorf("insert daily summary: %w", err)
		}
		return s.DailySummaryByDate(ctx, userID, date)
	}

	generationCount := existing.GenerationCount
	if input.IncrementCount {
		generationCount++
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE daily_summary_records
		SET draft_text = ?, manual_note = ?, options_json = ?, generation_source = ?,
			generation_count = ?, context_json = ?, updated_at = ?
		WHERE user_id = ? AND summary_date = ? AND status = 'draft'
	`, strings.TrimSpace(input.DraftText), strings.TrimSpace(input.ManualNote), string(optionsJSON),
		generationSource, generationCount, contextJSON, now, userID, date); err != nil {
		return nil, fmt.Errorf("update daily summary draft: %w", err)
	}
	return s.DailySummaryByDate(ctx, userID, date)
}

func (s *Store) ApproveDailySummary(ctx context.Context, userID string, date string, approvedText string) (*DailySummaryRecord, error) {
	existing, err := s.DailySummaryByDate(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	if existing.Status == DailySummaryApproved {
		return existing, nil
	}

	approvedText = strings.TrimSpace(approvedText)
	if approvedText == "" {
		approvedText = strings.TrimSpace(existing.DraftText)
	}
	if approvedText == "" {
		return nil, validationError(ErrInvalidTimeEntryInput, "draftText", "required", "draft text is required before approval")
	}

	now := nowString()
	if _, err := s.db.ExecContext(ctx, `
		UPDATE daily_summary_records
		SET status = 'approved', approved_text = ?, draft_text = ?, approved_at = ?, updated_at = ?
		WHERE user_id = ? AND summary_date = ?
	`, approvedText, approvedText, now, now, userID, date); err != nil {
		return nil, fmt.Errorf("approve daily summary: %w", err)
	}
	return s.DailySummaryByDate(ctx, userID, date)
}

func (s *Store) ReopenDailySummary(ctx context.Context, userID string, date string) (*DailySummaryRecord, error) {
	existing, err := s.DailySummaryByDate(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	if existing.Status != DailySummaryApproved {
		return existing, nil
	}

	now := nowString()
	if _, err := s.db.ExecContext(ctx, `
		UPDATE daily_summary_records
		SET status = 'draft', draft_text = ?, approved_at = NULL, updated_at = ?
		WHERE user_id = ? AND summary_date = ?
	`, existing.ApprovedText, now, userID, date); err != nil {
		return nil, fmt.Errorf("reopen daily summary: %w", err)
	}
	return s.DailySummaryByDate(ctx, userID, date)
}

func scanDailySummaryRecord(scanner interface{ Scan(dest ...any) error }) (DailySummaryRecord, error) {
	var record DailySummaryRecord
	var optionsJSON string
	var contextJSON string
	if err := scanner.Scan(
		&record.ID,
		&record.Date,
		&record.Status,
		&record.DraftText,
		&record.ApprovedText,
		&record.ManualNote,
		&optionsJSON,
		&record.GenerationSource,
		&record.GenerationCount,
		&contextJSON,
		&record.ApprovedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return DailySummaryRecord{}, err
	}
	if err := json.Unmarshal([]byte(optionsJSON), &record.Options); err != nil {
		return DailySummaryRecord{}, fmt.Errorf("decode summary options: %w", err)
	}
	record.ContextJSON = contextJSON
	return record, nil
}
