package store

import (
	"context"
	"testing"
)

func TestDailySummaryApproveAndReopen(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	date := "2026-07-12"
	draft, err := st.UpsertDailySummaryDraft(ctx, user.ID, date, DailySummaryRecordInput{
		DraftText:        "12/7:\nResumen de hoy:\nPor la mañana avancé con leotime.",
		GenerationSource: "template",
		IncrementCount:   true,
	})
	if err != nil {
		t.Fatalf("upsert draft: %v", err)
	}
	if draft.Status != DailySummaryDraft {
		t.Fatalf("expected draft, got %s", draft.Status)
	}

	approved, err := st.ApproveDailySummary(ctx, user.ID, date, "", "", draft.DraftText)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != DailySummaryApproved || approved.ApprovedText == "" {
		t.Fatalf("expected approved record, got %+v", approved)
	}

	if _, err := st.UpsertDailySummaryDraft(ctx, user.ID, date, DailySummaryRecordInput{
		DraftText: "should fail",
	}); err == nil {
		t.Fatal("expected approved lock on draft update")
	}

	reopened, err := st.ReopenDailySummary(ctx, user.ID, date, "", "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Status != DailySummaryDraft {
		t.Fatalf("expected draft after reopen, got %s", reopened.Status)
	}
}
