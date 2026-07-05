package store

import (
	"context"
	"testing"
	"time"
)

func TestBuildDashboardStatsAggregatesWeekAndRecent(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	today := dateOnlyUTC(time.Now().UTC())
	recentStart := today.Add(9 * time.Hour).Format(time.RFC3339)
	recentEnd := today.Add(10 * time.Hour).Format(time.RFC3339)
	weekStart := startOfWeekUTC(today, time.Monday)
	olderDay := weekStart.Format("2006-01-02")
	olderStart := olderDay + "T08:00:00Z"
	olderEnd := olderDay + "T09:30:00Z"

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:                   "Acme",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 10000,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	project, err := st.CreateProject(ctx, user.ID, ProjectInput{
		ClientID: client.ID,
		Name:     "Portal Web",
		Color:    "#2563eb",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "Recent work",
		StartedAt:   recentStart,
		EndedAt:     recentEnd,
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create recent entry: %v", err)
	}
	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "Older work",
		StartedAt:   olderStart,
		EndedAt:     olderEnd,
		Billable:    false,
	})
	if err != nil {
		t.Fatalf("create older entry: %v", err)
	}

	stats, err := st.BuildDashboardStats(ctx, user.ID, "")
	if err != nil {
		t.Fatalf("build dashboard stats: %v", err)
	}

	if len(stats.RecentEntries) == 0 || stats.RecentEntries[0].Description != "Recent work" {
		t.Fatalf("unexpected recent entries: %+v", stats.RecentEntries)
	}
	if len(stats.LastSevenDays) != 7 {
		t.Fatalf("expected 7 day summaries, got %d", len(stats.LastSevenDays))
	}
	if len(stats.WeekDays) != 7 {
		t.Fatalf("expected 7 week days, got %d", len(stats.WeekDays))
	}
	if stats.WeekSpentSeconds <= 0 {
		t.Fatalf("expected week spent seconds > 0, got %d", stats.WeekSpentSeconds)
	}
	if stats.WeekBillableSeconds <= 0 || stats.WeekBillableMinor <= 0 {
		t.Fatalf("expected billable totals, got seconds=%d minor=%d", stats.WeekBillableSeconds, stats.WeekBillableMinor)
	}
	if len(stats.ProjectBreakdown) == 0 {
		t.Fatalf("expected project breakdown")
	}
	if len(stats.ActivityHeatmap) == 0 {
		t.Fatalf("expected activity heatmap days")
	}
	if stats.ActivityMonth == "" {
		t.Fatalf("expected activity month")
	}
}

func TestHeatmapLevelThresholds(t *testing.T) {
	if heatmapLevel(0) != 0 {
		t.Fatal("expected level 0 for zero seconds")
	}
	if heatmapLevel(3600) != 1 {
		t.Fatal("expected level 1 for one hour")
	}
	if heatmapLevel(5*3600) != 3 {
		t.Fatal("expected level 3 for five hours")
	}
	if heatmapLevel(7*3600) != 4 {
		t.Fatal("expected level 4 for seven hours")
	}
}
