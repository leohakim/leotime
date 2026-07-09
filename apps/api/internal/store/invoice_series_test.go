package store

import (
	"context"
	"testing"
	"time"
)

func TestFormatInvoiceNumber(t *testing.T) {
	issueTime := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		pattern  string
		sequence int
		want     string
	}{
		{"year padded", "{YYYY}-{SEQ:04}", 9, "2026-0009"},
		{"invoice prefix", "INV-{YYYY}-{SEQ:03}", 12, "INV-2026-012"},
		{"short year", "{YY}/{SEQ}", 15, "26/15"},
	}
	for _, tc := range cases {
		got, err := FormatInvoiceNumber(tc.pattern, issueTime, tc.sequence)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestInvoiceSeriesCRUD(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	series, err := st.CreateInvoiceSeries(ctx, user.ID, InvoiceSeriesInput{
		Code:    "CRAFT",
		Name:    "Main invoices",
		Pattern: "{YYYY}-{SEQ:04}",
		Default: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	if series.Code != "CRAFT" || !series.Default || series.NextSequence != 1 {
		t.Fatalf("unexpected series: %+v", series)
	}

	list, err := st.ListInvoiceSeries(ctx, user.ID)
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected two series, got %d", len(list))
	}

	updated, err := st.UpdateInvoiceSeries(ctx, user.ID, series.ID, InvoiceSeriesInput{
		Name:         "Primary",
		NextSequence: intPtr(5),
	})
	if err != nil {
		t.Fatalf("update series: %v", err)
	}
	if updated.Name != "Primary" || updated.NextSequence != 5 {
		t.Fatalf("unexpected updated series: %+v", updated)
	}
}

func TestNextInvoiceNumberTxIncrementsAndRollsBack(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	series, err := st.CreateInvoiceSeries(ctx, user.ID, InvoiceSeriesInput{
		Code:    "ALT",
		Name:    "Alt",
		Pattern: "{YYYY}-{SEQ:04}",
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	issueTime := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	tx1, err := st.DB().BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}
	first, seq1, err := st.NextInvoiceNumberTx(ctx, tx1, user.ID, series.ID, issueTime)
	if err != nil {
		t.Fatalf("next number tx1: %v", err)
	}
	if first != "2026-0001" || seq1 != 1 {
		t.Fatalf("first number: got %q seq %d", first, seq1)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatalf("commit tx1: %v", err)
	}

	tx2, err := st.DB().BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	second, seq2, err := st.NextInvoiceNumberTx(ctx, tx2, user.ID, series.ID, issueTime)
	if err != nil {
		t.Fatalf("next number tx2: %v", err)
	}
	if second != "2026-0002" || seq2 != 2 {
		t.Fatalf("second number: got %q seq %d", second, seq2)
	}
	if err := tx2.Rollback(); err != nil {
		t.Fatalf("rollback tx2: %v", err)
	}

	tx3, err := st.DB().BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx3: %v", err)
	}
	third, seq3, err := st.NextInvoiceNumberTx(ctx, tx3, user.ID, series.ID, issueTime)
	if err != nil {
		t.Fatalf("next number tx3: %v", err)
	}
	if third != "2026-0002" || seq3 != 2 {
		t.Fatalf("after rollback expected 2026-0002 seq 2, got %q seq %d", third, seq3)
	}
	if err := tx3.Commit(); err != nil {
		t.Fatalf("commit tx3: %v", err)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}
