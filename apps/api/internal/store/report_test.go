package store

import (
	"context"
	"testing"
)

func TestBuildTimeReportGroupsByProject(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	project, err := st.CreateProject(ctx, user.ID, ProjectInput{Name: "Portal Web", Color: "#2563eb"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ProjectID:   project.ID,
		Description: "Design",
		StartedAt:   "2026-07-01T08:00:00Z",
		EndedAt:     "2026-07-01T09:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create entry A: %v", err)
	}
	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ProjectID:   project.ID,
		Description: "Build",
		StartedAt:   "2026-07-02T10:00:00Z",
		EndedAt:     "2026-07-02T11:30:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create entry B: %v", err)
	}

	report, err := st.BuildTimeReport(ctx, user.ID, TimeReportOptions{
		From:    "2026-07-01T00:00:00Z",
		To:      "2026-07-31T23:59:59Z",
		GroupBy: "project",
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if report.EntryCount != 2 || report.TotalSeconds != 9000 {
		t.Fatalf("unexpected totals: %+v", report)
	}
	if len(report.Groups) != 1 || report.Groups[0].Label != "Portal Web" || report.Groups[0].TotalSeconds != 9000 {
		t.Fatalf("unexpected groups: %+v", report.Groups)
	}
	if report.Groups[0].ProjectColor != "#2563eb" {
		t.Fatalf("expected project color in grouped report: %+v", report.Groups[0])
	}
	if len(report.Entries) != 0 {
		t.Fatalf("expected no entries in summary mode")
	}
}

func TestBuildTimeReportDetailedIncludesTimestamps(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	_, err := st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		Description: "Solo work",
		StartedAt:   "2026-07-05T12:00:00Z",
		EndedAt:     "2026-07-05T13:00:00Z",
		Billable:    false,
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	report, err := st.BuildTimeReport(ctx, user.ID, TimeReportOptions{
		From:              "2026-07-01T00:00:00Z",
		To:                "2026-07-31T23:59:59Z",
		IncludeTimestamps: true,
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if len(report.Entries) != 1 || report.Entries[0].StartedAt == "" || report.Entries[0].EndedAt == "" {
		t.Fatalf("expected detailed entry timestamps: %+v", report.Entries)
	}
}

func TestBuildTimeReportBillableOnlyFilter(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:                   "Billable Client",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 5000,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		Description: "Billable",
		StartedAt:   "2026-07-01T08:00:00Z",
		EndedAt:     "2026-07-01T09:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create billable entry: %v", err)
	}
	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		Description: "Internal",
		StartedAt:   "2026-07-01T10:00:00Z",
		EndedAt:     "2026-07-01T10:30:00Z",
		Billable:    false,
	})
	if err != nil {
		t.Fatalf("create non-billable entry: %v", err)
	}

	report, err := st.BuildTimeReport(ctx, user.ID, TimeReportOptions{
		From:         "2026-07-01T00:00:00Z",
		To:           "2026-07-31T23:59:59Z",
		GroupBy:      "day",
		BillableOnly: true,
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if report.EntryCount != 1 || report.TotalSeconds != 3600 {
		t.Fatalf("unexpected billable-only totals: %+v", report)
	}
}
