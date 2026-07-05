package store

import (
	"context"
	"testing"
)

func TestCreateInvoiceDraftFromTime(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:                   "Acme Corp",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 10000,
		TaxID:                  "B12345678",
		BillingAddress:         "Madrid",
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
		Description: "Design sprint",
		StartedAt:   "2026-07-01T08:00:00Z",
		EndedAt:     "2026-07-01T10:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	invoice, err := st.CreateInvoiceDraftFromTime(ctx, user.ID, InvoiceDraftFromTimeInput{
		ClientID:           client.ID,
		From:               "2026-07-01T00:00:00Z",
		To:                 "2026-07-31T23:59:59Z",
		TaxRateBasisPoints: 2100,
	})
	if err != nil {
		t.Fatalf("create invoice draft: %v", err)
	}

	if invoice.Status != "draft" || invoice.Currency != "EUR" {
		t.Fatalf("unexpected invoice header: %+v", invoice)
	}
	if len(invoice.Lines) != 1 {
		t.Fatalf("expected one line, got %+v", invoice.Lines)
	}
	if invoice.Lines[0].QuantityMinutes != 120 || invoice.Lines[0].UnitRateMinor != 10000 {
		t.Fatalf("unexpected line: %+v", invoice.Lines[0])
	}
	if invoice.SubtotalMinor != 20000 || invoice.TaxMinor != 4200 || invoice.TotalMinor != 24200 {
		t.Fatalf("unexpected totals: subtotal=%d tax=%d total=%d", invoice.SubtotalMinor, invoice.TaxMinor, invoice.TotalMinor)
	}
	if invoice.ClientName != "Acme Corp" || invoice.SellerName != user.Name {
		t.Fatalf("unexpected frozen fields: %+v", invoice)
	}
}

func TestCreateInvoiceDraftSkipsAlreadyInvoicedEntries(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:                   "Repeat Client",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 5000,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		Description: "First block",
		StartedAt:   "2026-07-02T08:00:00Z",
		EndedAt:     "2026-07-02T09:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	first, err := st.CreateInvoiceDraftFromTime(ctx, user.ID, InvoiceDraftFromTimeInput{
		ClientID: client.ID,
		From:     "2026-07-01T00:00:00Z",
		To:       "2026-07-31T23:59:59Z",
	})
	if err != nil {
		t.Fatalf("create first invoice: %v", err)
	}
	if len(first.Lines) != 1 {
		t.Fatalf("expected one line on first invoice")
	}

	_, err = st.CreateInvoiceDraftFromTime(ctx, user.ID, InvoiceDraftFromTimeInput{
		ClientID: client.ID,
		From:     "2026-07-01T00:00:00Z",
		To:       "2026-07-31T23:59:59Z",
	})
	if err == nil {
		t.Fatal("expected error when no uninvoiced entries remain")
	}
}

func TestUpdateInvoiceStatusSetsIssuedAt(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:                   "Status Client",
		DefaultCurrency:        "USD",
		DefaultHourlyRateMinor: 7500,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		Description: "Support",
		StartedAt:   "2026-07-03T12:00:00Z",
		EndedAt:     "2026-07-03T13:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	invoice, err := st.CreateInvoiceDraftFromTime(ctx, user.ID, InvoiceDraftFromTimeInput{
		ClientID: client.ID,
		From:     "2026-07-01T00:00:00Z",
		To:       "2026-07-31T23:59:59Z",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	issued, err := st.UpdateInvoiceStatus(ctx, user.ID, invoice.ID, "issued")
	if err != nil {
		t.Fatalf("issue invoice: %v", err)
	}
	if issued.Status != "issued" || issued.IssuedAt == "" {
		t.Fatalf("expected issued invoice with issuedAt, got %+v", issued)
	}

	_, err = st.UpdateInvoice(ctx, user.ID, invoice.ID, InvoiceUpdateInput{
		Notes: strPtr("Updated notes"),
	})
	if err == nil {
		t.Fatal("expected non-editable issued invoice")
	}
}

func strPtr(value string) *string {
	return &value
}
