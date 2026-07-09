package billing

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestRenderPreviewHTML(t *testing.T) {
	renderer := NewHTMLRenderer()
	snapshot := sampleSnapshot()

	html, err := renderer.RenderPreviewHTML(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("render preview html: %v", err)
	}
	body := string(html)
	for _, want := range []string{
		"Invoice #",
		"Work Protocol #",
		"Acme Corp",
		"Portal Web",
		"Description",
		"Rate Hour",
		"Qty",
		"Amount",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected preview html to contain %q", want)
		}
	}
}

func TestRenderPDFs(t *testing.T) {
	renderer := NewPDFRenderer()
	outputDir := t.TempDir()

	paths, err := renderer.RenderPDFs(context.Background(), sampleSnapshot(), outputDir)
	if err != nil {
		t.Fatalf("render pdfs: %v", err)
	}

	for _, path := range []string{paths.InvoicePath, paths.WorkProtocolPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read pdf %s: %v", path, err)
		}
		if len(data) == 0 || string(data[:4]) != "%PDF" {
			t.Fatalf("expected non-empty pdf at %s", path)
		}
	}
}

func sampleSnapshot() DocumentSnapshot {
	return DocumentSnapshot{
		Version: SnapshotVersion,
		Invoice: InvoiceSnapshot{
			Number:        "2026-0009",
			Currency:      "EUR",
			IssuedAt:      "2026-07-08",
			SellerName:    "Seller LLC",
			SellerTaxID:   "TAX-1",
			SellerAddress: "Madrid",
			ClientName:    "Acme Corp",
			ClientTaxID:   "B123",
			ClientAddress: "Barcelona",
			SubtotalMinor: 20000,
			TaxMinor:      4200,
			TotalMinor:    24200,
			Lines: []InvoiceLineSnapshot{
				{Description: "Portal Web — Design", QuantityMinutes: 120, UnitRateMinor: 10000, SubtotalMinor: 20000},
			},
		},
		WorkProtocol: WorkProtocolSnapshot{
			Number: "2026-0009",
			Detail: WorkProtocolStandard,
			Rows: []WorkProtocolDayRow{
				{Date: "2026-07-01", Hours: "2.00", Items: []string{"Portal Web / Design — 2.00"}},
			},
		},
	}
}
