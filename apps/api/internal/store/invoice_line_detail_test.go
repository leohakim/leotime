package store

import (
	"testing"
)

func TestMergeInvoiceLinesSummaryUsesSingleDescription(t *testing.T) {
	lines := []InvoiceLine{
		{TimeEntryID: "ten_1", Description: "Portal Web — Design", QuantityMinutes: 90, UnitRateMinor: 10000, SubtotalMinor: 15000, TaxRateBasisPoints: 2100},
		{TimeEntryID: "ten_2", Description: "Portal Web — Build", QuantityMinutes: 60, UnitRateMinor: 10000, SubtotalMinor: 10000, TaxRateBasisPoints: 2100},
	}
	entries := map[string]TimeEntry{
		"ten_1": {ID: "ten_1", ProjectName: "Portal Web", TaskName: "Design"},
		"ten_2": {ID: "ten_2", ProjectName: "Portal Web", TaskName: "Build"},
	}

	merged := mergeInvoiceLinesForDisplay(lines, entries, InvoiceLineDetailSummary, "es")
	if len(merged) != 1 {
		t.Fatalf("expected one summary line, got %+v", merged)
	}
	if merged[0].Description != "Horas de Ingeniería de Software" {
		t.Fatalf("unexpected summary description: %q", merged[0].Description)
	}
	if merged[0].QuantityMinutes != 180 {
		t.Fatalf("expected 3 whole hours, got %d minutes", merged[0].QuantityMinutes)
	}
}

func TestMergeInvoiceLinesByProjectGroupsProjects(t *testing.T) {
	lines := []InvoiceLine{
		{TimeEntryID: "ten_1", QuantityMinutes: 120, UnitRateMinor: 10000, SubtotalMinor: 20000, TaxRateBasisPoints: 2100},
		{TimeEntryID: "ten_2", QuantityMinutes: 60, UnitRateMinor: 10000, SubtotalMinor: 10000, TaxRateBasisPoints: 2100},
	}
	entries := map[string]TimeEntry{
		"ten_1": {ID: "ten_1", ProjectID: "prj_1", ProjectName: "Portal Web"},
		"ten_2": {ID: "ten_2", ProjectID: "prj_2", ProjectName: "API"},
	}

	merged := mergeInvoiceLinesForDisplay(lines, entries, InvoiceLineDetailByProject, "es")
	if len(merged) != 2 {
		t.Fatalf("expected two project lines, got %+v", merged)
	}
	descriptions := map[string]bool{}
	for _, line := range merged {
		descriptions[line.Description] = true
	}
	if !descriptions["Portal Web"] || !descriptions["API"] {
		t.Fatalf("unexpected project descriptions: %+v", merged)
	}
}

func TestMergeInvoiceLinesDoubleMergeCollapsesProjects(t *testing.T) {
	lines := []InvoiceLine{
		{TimeEntryID: "ten_1", QuantityMinutes: 120, UnitRateMinor: 10000, SubtotalMinor: 20000, TaxRateBasisPoints: 2100},
		{TimeEntryID: "ten_2", QuantityMinutes: 60, UnitRateMinor: 10000, SubtotalMinor: 10000, TaxRateBasisPoints: 2100},
	}
	entries := map[string]TimeEntry{
		"ten_1": {ID: "ten_1", ProjectID: "prj_1", ProjectName: "API migration"},
		"ten_2": {ID: "ten_2", ProjectID: "prj_2", ProjectName: "Website redesign"},
	}

	first := mergeInvoiceLinesForDisplay(lines, entries, InvoiceLineDetailByProject, "en")
	second := mergeInvoiceLinesForDisplay(first, entries, InvoiceLineDetailByProject, "en")
	if len(second) != 1 {
		t.Fatalf("expected double merge bug to collapse to one line, got %+v", second)
	}
	if second[0].Description != "Billable time" {
		t.Fatalf("expected fallback description after double merge, got %q", second[0].Description)
	}
}

func TestRoundMinutesToWholeHours(t *testing.T) {
	if got := roundMinutesToWholeHours(90); got != 120 {
		t.Fatalf("expected 120 minutes for 90 raw minutes, got %d", got)
	}
	if got := roundMinutesToWholeHours(20); got != 60 {
		t.Fatalf("expected minimum one hour, got %d", got)
	}
}

func TestTotalInvoiceQuantityMinutes(t *testing.T) {
	total := TotalInvoiceQuantityMinutes([]InvoiceLine{
		{QuantityMinutes: 120},
		{QuantityMinutes: 180},
	})
	if total != 300 {
		t.Fatalf("expected 300 total minutes, got %d", total)
	}
}
