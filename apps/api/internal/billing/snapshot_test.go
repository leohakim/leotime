package billing

import (
	"strings"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestBuildDocumentSnapshotDetailLevels(t *testing.T) {
	invoice := &store.Invoice{
		InvoiceNumber:      "2026-0009",
		Status:             "draft",
		Currency:           "EUR",
		SellerName:         "Seller",
		ClientName:         "Client",
		PeriodFrom:         "2026-07-01T00:00:00Z",
		PeriodTo:           "2026-07-31T23:59:59Z",
		SubtotalMinor:      20000,
		TaxMinor:           4200,
		TotalMinor:         24200,
		WorkProtocolDetail: "standard",
		Lines: []store.InvoiceLine{
			{Description: "Portal Web — Design", QuantityMinutes: 120, UnitRateMinor: 10000, SubtotalMinor: 20000},
		},
	}

	entries := []store.TimeEntry{
		{
			ProjectName:     "Portal Web",
			TaskName:        "Design",
			Description:     "Wireframes",
			StartedAt:       "2026-07-01T08:00:00Z",
			EndedAt:         "2026-07-01T10:00:00Z",
			DurationSeconds: 7200,
			Tags:            []store.TimeEntryTag{{Name: "UX"}},
		},
		{
			ProjectName:     "Portal Web",
			TaskName:        "Build",
			Description:     "Components",
			StartedAt:       "2026-07-02T09:00:00Z",
			EndedAt:         "2026-07-02T11:00:00Z",
			DurationSeconds: 7200,
		},
	}

	summaryInvoice := *invoice
	summaryInvoice.WorkProtocolDetail = "summary"
	summary, err := BuildDocumentSnapshot(&summaryInvoice, entries, SnapshotOptions{})
	if err != nil {
		t.Fatalf("summary snapshot: %v", err)
	}
	if len(summary.WorkProtocol.Rows) != 2 {
		t.Fatalf("expected two summary rows, got %d", len(summary.WorkProtocol.Rows))
	}
	if len(summary.WorkProtocol.Rows[0].Items) == 0 || summary.WorkProtocol.Rows[0].Items[0] != "Portal Web" {
		t.Fatalf("unexpected summary items: %+v", summary.WorkProtocol.Rows[0])
	}

	standard, err := BuildDocumentSnapshot(invoice, entries, SnapshotOptions{})
	if err != nil {
		t.Fatalf("standard snapshot: %v", err)
	}
	if len(standard.WorkProtocol.Rows[0].Items) == 0 {
		t.Fatalf("expected standard bullets, got %+v", standard.WorkProtocol.Rows[0])
	}

	invoice.WorkProtocolDetail = "detailed"
	detailed, err := BuildDocumentSnapshot(invoice, entries, SnapshotOptions{})
	if err != nil {
		t.Fatalf("detailed snapshot: %v", err)
	}
	if !strings.Contains(detailed.WorkProtocol.Rows[0].Items[0], "UX") {
		t.Fatalf("expected tags in detailed row, got %+v", detailed.WorkProtocol.Rows[0].Items)
	}
	if !strings.Contains(detailed.WorkProtocol.Rows[0].Items[0], "08:00") {
		t.Fatalf("expected time range in detailed row, got %+v", detailed.WorkProtocol.Rows[0].Items)
	}
	if strings.Contains(standard.WorkProtocol.Rows[0].Items[0], "Wireframes") {
		t.Fatalf("standard row should not include entry descriptions, got %+v", standard.WorkProtocol.Rows[0].Items)
	}

	preview, err := BuildDocumentSnapshot(invoice, entries, SnapshotOptions{
		Preview:    true,
		IssueAt:    time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
		SeriesCode: "MAIN",
	})
	if err != nil {
		t.Fatalf("preview snapshot: %v", err)
	}
	if !strings.HasPrefix(preview.Invoice.Number, "PREVIEW-") {
		t.Fatalf("expected preview number, got %q", preview.Invoice.Number)
	}
}

func TestBuildDocumentSnapshotByProjectGroupsLines(t *testing.T) {
	invoice := &store.Invoice{
		InvoiceNumber:      "2026-0010",
		Status:             "draft",
		Currency:           "EUR",
		SellerName:         "Seller",
		ClientName:         "ACME Corp",
		PeriodFrom:         "2026-05-01T00:00:00Z",
		PeriodTo:           "2026-05-31T23:59:59Z",
		InvoiceLineDetail:  store.InvoiceLineDetailByProject,
		WorkProtocolDetail: "standard",
		Lines: []store.InvoiceLine{
			{TimeEntryID: "ten_1", QuantityMinutes: 120, UnitRateMinor: 7500, SubtotalMinor: 15000, TaxRateBasisPoints: 0},
			{TimeEntryID: "ten_2", QuantityMinutes: 180, UnitRateMinor: 7500, SubtotalMinor: 22500, TaxRateBasisPoints: 0},
			{TimeEntryID: "ten_3", QuantityMinutes: 60, UnitRateMinor: 8000, SubtotalMinor: 8000, TaxRateBasisPoints: 0},
		},
	}
	entries := []store.TimeEntry{
		{ID: "ten_1", ProjectID: "prj_1", ProjectName: "API migration", StartedAt: "2026-05-01T09:00:00Z", EndedAt: "2026-05-01T11:00:00Z", DurationSeconds: 7200},
		{ID: "ten_2", ProjectID: "prj_2", ProjectName: "Website redesign", StartedAt: "2026-05-02T09:00:00Z", EndedAt: "2026-05-02T12:00:00Z", DurationSeconds: 10800},
		{ID: "ten_3", ProjectID: "prj_1", ProjectName: "API migration", StartedAt: "2026-05-03T10:00:00Z", EndedAt: "2026-05-03T11:00:00Z", DurationSeconds: 3600},
	}

	snapshot, err := BuildDocumentSnapshot(invoice, entries, SnapshotOptions{Locale: "en"})
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	if len(snapshot.Invoice.Lines) != 2 {
		t.Fatalf("expected two project lines, got %+v", snapshot.Invoice.Lines)
	}

	descriptions := map[string]bool{}
	for _, line := range snapshot.Invoice.Lines {
		descriptions[line.Description] = true
	}
	if !descriptions["API migration"] || !descriptions["Website redesign"] {
		t.Fatalf("unexpected invoice line descriptions: %+v", snapshot.Invoice.Lines)
	}
	if snapshot.Invoice.TotalQuantityMinutes != 360 {
		t.Fatalf("expected 360 total minutes (6 hours), got %d", snapshot.Invoice.TotalQuantityMinutes)
	}
}
