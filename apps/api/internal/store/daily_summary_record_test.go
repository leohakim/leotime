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

func TestListDailySummaryIndexFiltersByScopeAndRange(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	if _, err := st.UpsertDailySummaryDraft(ctx, user.ID, "2026-07-10", DailySummaryRecordInput{
		DraftText: "whole day",
	}); err != nil {
		t.Fatalf("upsert whole day: %v", err)
	}
	if _, err := st.UpsertDailySummaryDraft(ctx, user.ID, "2026-07-11", DailySummaryRecordInput{
		DraftText: "approved day",
	}); err != nil {
		t.Fatalf("upsert approved day draft: %v", err)
	}
	if _, err := st.ApproveDailySummary(ctx, user.ID, "2026-07-11", "", "", "approved day"); err != nil {
		t.Fatalf("approve day: %v", err)
	}
	if _, err := st.UpsertDailySummaryDraft(ctx, user.ID, "2026-07-12", DailySummaryRecordInput{
		DraftText:        "draft day",
		GenerationSource: "cursor",
		IncrementCount:   true,
	}); err != nil {
		t.Fatalf("upsert draft day: %v", err)
	}

	items, err := st.ListDailySummaryIndex(ctx, user.ID, "2026-07-01", "2026-07-31", "", "", false)
	if err != nil {
		t.Fatalf("list index: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].Date != "2026-07-10" || items[0].Status != DailySummaryDraft {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].Status != DailySummaryApproved {
		t.Fatalf("expected approved on second item, got %+v", items[1])
	}
	if items[2].GenerationSource != "cursor" {
		t.Fatalf("expected cursor source on third item, got %+v", items[2])
	}
}

func TestListDailySummaryIndexAllScopesIgnoresClientFilter(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Acme"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if _, err := st.UpsertDailySummaryDraft(ctx, user.ID, "2026-07-15", DailySummaryRecordInput{
		DraftText: "whole day",
	}); err != nil {
		t.Fatalf("upsert whole day: %v", err)
	}
	if _, err := st.UpsertDailySummaryDraft(ctx, user.ID, "2026-07-15", DailySummaryRecordInput{
		DraftText: "client scoped",
		Options:   DailySummaryOptions{ClientID: client.ID},
	}); err != nil {
		t.Fatalf("upsert client scoped: %v", err)
	}

	scoped, err := st.ListDailySummaryIndex(ctx, user.ID, "2026-07-01", "2026-07-31", "", "", false)
	if err != nil {
		t.Fatalf("list scoped index: %v", err)
	}
	if len(scoped) != 1 {
		t.Fatalf("expected 1 scoped item, got %d", len(scoped))
	}

	all, err := st.ListDailySummaryIndex(ctx, user.ID, "2026-07-01", "2026-07-31", "", "", true)
	if err != nil {
		t.Fatalf("list all scopes index: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 items across scopes, got %d", len(all))
	}
}
