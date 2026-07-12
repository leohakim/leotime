package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type DailySummaryAIRun struct {
	ID               string  `json:"id"`
	SummaryDate      string  `json:"summaryDate"`
	ClientID         string  `json:"clientId"`
	ProjectID        string  `json:"projectId"`
	RecordID         string  `json:"recordId,omitempty"`
	ModelID          string  `json:"modelId"`
	Source           string  `json:"source"`
	InputTokens      int     `json:"inputTokens"`
	OutputTokens     int     `json:"outputTokens"`
	CacheReadTokens  int     `json:"cacheReadTokens"`
	CacheWriteTokens int     `json:"cacheWriteTokens"`
	TotalTokens      int     `json:"totalTokens"`
	EstimatedCostUSD float64 `json:"estimatedCostUsd"`
	CreatedAt        string  `json:"createdAt"`
}

type DailySummaryAIRunInput struct {
	SummaryDate      string
	ClientID         string
	ProjectID        string
	RecordID         string
	ModelID          string
	Source           string
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	TotalTokens      int
}

type DailySummaryAIUsageSummary struct {
	From             string  `json:"from"`
	To               string  `json:"to"`
	RunCount         int     `json:"runCount"`
	InputTokens      int     `json:"inputTokens"`
	OutputTokens     int     `json:"outputTokens"`
	CacheReadTokens  int     `json:"cacheReadTokens"`
	CacheWriteTokens int     `json:"cacheWriteTokens"`
	TotalTokens      int     `json:"totalTokens"`
	EstimatedCostUSD float64 `json:"estimatedCostUsd"`
	CostPerMillion   float64 `json:"costPerMillionUsd"`
}

func EstimateTokenCostUSD(totalTokens int, costPerMillion float64) float64 {
	if totalTokens <= 0 || costPerMillion <= 0 {
		return 0
	}
	return float64(totalTokens) / 1_000_000 * costPerMillion
}

func (s *Store) InsertDailySummaryAIRun(ctx context.Context, userID string, input DailySummaryAIRunInput, costPerMillion float64) (*DailySummaryAIRun, error) {
	input.SummaryDate = strings.TrimSpace(input.SummaryDate)
	if _, err := time.Parse("2006-01-02", input.SummaryDate); err != nil {
		return nil, validationError(ErrInvalidTimeEntryInput, "date", "invalid", "date must be YYYY-MM-DD")
	}
	input.ClientID, input.ProjectID = NormalizeDailySummaryScope(input.ClientID, input.ProjectID)
	if input.TotalTokens == 0 {
		input.TotalTokens = input.InputTokens + input.OutputTokens + input.CacheReadTokens + input.CacheWriteTokens
	}

	runID, err := newID("dsr")
	if err != nil {
		return nil, err
	}
	now := nowString()
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "cursor"
	}
	modelID := strings.TrimSpace(input.ModelID)
	if modelID == "" {
		modelID = "composer-2.5"
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO daily_summary_ai_runs (
			id, user_id, summary_date, client_id, project_id, record_id, model_id, source,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, total_tokens, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, runID, userID, input.SummaryDate, input.ClientID, input.ProjectID, strings.TrimSpace(input.RecordID),
		modelID, source, input.InputTokens, input.OutputTokens, input.CacheReadTokens, input.CacheWriteTokens,
		input.TotalTokens, now); err != nil {
		return nil, fmt.Errorf("insert daily summary ai run: %w", err)
	}

	return &DailySummaryAIRun{
		ID:               runID,
		SummaryDate:      input.SummaryDate,
		ClientID:         input.ClientID,
		ProjectID:        input.ProjectID,
		RecordID:         strings.TrimSpace(input.RecordID),
		ModelID:          modelID,
		Source:           source,
		InputTokens:      input.InputTokens,
		OutputTokens:     input.OutputTokens,
		CacheReadTokens:  input.CacheReadTokens,
		CacheWriteTokens: input.CacheWriteTokens,
		TotalTokens:      input.TotalTokens,
		EstimatedCostUSD: EstimateTokenCostUSD(input.TotalTokens, costPerMillion),
		CreatedAt:        now,
	}, nil
}

func (s *Store) ListDailySummaryAIRuns(ctx context.Context, userID, from, to string) ([]DailySummaryAIRun, float64, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return nil, 0, validationError(ErrInvalidTimeEntryInput, "date", "required", "from and to are required")
	}

	costPerMillion, err := s.cursorCostPerMillionUSD(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, summary_date, client_id, project_id, COALESCE(record_id, ''), model_id, source,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, total_tokens, created_at
		FROM daily_summary_ai_runs
		WHERE user_id = ? AND summary_date >= ? AND summary_date <= ?
		ORDER BY created_at DESC
	`, userID, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("list daily summary ai runs: %w", err)
	}
	defer rows.Close()

	runs := make([]DailySummaryAIRun, 0)
	for rows.Next() {
		var run DailySummaryAIRun
		if err := rows.Scan(
			&run.ID, &run.SummaryDate, &run.ClientID, &run.ProjectID, &run.RecordID, &run.ModelID, &run.Source,
			&run.InputTokens, &run.OutputTokens, &run.CacheReadTokens, &run.CacheWriteTokens, &run.TotalTokens, &run.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan daily summary ai run: %w", err)
		}
		run.EstimatedCostUSD = EstimateTokenCostUSD(run.TotalTokens, costPerMillion)
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate daily summary ai runs: %w", err)
	}
	return runs, costPerMillion, nil
}

func (s *Store) SummarizeDailySummaryAIUsage(ctx context.Context, userID, from, to string) (*DailySummaryAIUsageSummary, error) {
	runs, costPerMillion, err := s.ListDailySummaryAIRuns(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	summary := &DailySummaryAIUsageSummary{
		From:           from,
		To:             to,
		RunCount:       len(runs),
		CostPerMillion: costPerMillion,
	}
	for _, run := range runs {
		summary.InputTokens += run.InputTokens
		summary.OutputTokens += run.OutputTokens
		summary.CacheReadTokens += run.CacheReadTokens
		summary.CacheWriteTokens += run.CacheWriteTokens
		summary.TotalTokens += run.TotalTokens
	}
	summary.EstimatedCostUSD = EstimateTokenCostUSD(summary.TotalTokens, costPerMillion)
	return summary, nil
}

func (s *Store) cursorCostPerMillionUSD(ctx context.Context, userID string) (float64, error) {
	var value float64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(cursor_cost_per_million_usd, 2.0)
		FROM app_settings
		WHERE user_id = ?
	`, userID).Scan(&value)
	if err != nil {
		return 2.0, nil
	}
	if value <= 0 {
		return 2.0, nil
	}
	return value, nil
}
