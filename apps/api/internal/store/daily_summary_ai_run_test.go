package store

import (
	"context"
	"testing"
)

func TestInsertAndSummarizeDailySummaryAIRuns(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	run, err := st.InsertDailySummaryAIRun(ctx, user.ID, DailySummaryAIRunInput{
		SummaryDate:  "2026-07-12",
		ModelID:      "composer-2.5",
		Source:       "cursor",
		InputTokens:  1200,
		OutputTokens: 400,
		TotalTokens:  1600,
	}, 2.0)
	if err != nil {
		t.Fatalf("insert ai run: %v", err)
	}
	if run.EstimatedCostUSD <= 0 {
		t.Fatalf("expected estimated cost, got %+v", run)
	}

	summary, err := st.SummarizeDailySummaryAIUsage(ctx, user.ID, "2026-07-01", "2026-07-31")
	if err != nil {
		t.Fatalf("summarize usage: %v", err)
	}
	if summary.TotalTokens != 1600 || summary.RunCount != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
