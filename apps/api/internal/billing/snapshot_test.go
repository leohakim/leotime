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
	if summary.WorkProtocol.Rows[0].ProjectNames != "Portal Web" {
		t.Fatalf("unexpected summary projects: %+v", summary.WorkProtocol.Rows[0])
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
