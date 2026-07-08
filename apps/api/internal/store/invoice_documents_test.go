package store

import (
	"context"
	"strings"
	"testing"
)

func TestInsertBillingDocumentValidation(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	invoice := createTestInvoiceDraft(t, ctx, st, user.ID)

	tx, err := st.DB().BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	validHash := strings.Repeat("a", 64)
	_, err = st.InsertBillingDocumentTx(ctx, tx, user.ID, BillingDocumentInput{
		InvoiceID:     invoice.ID,
		Kind:          "invoice_pdf",
		StoragePath:   "invoices/2026/MAIN/2026-0001/invoice.pdf",
		SHA256:        validHash,
		ByteSize:      1024,
		RenderVersion: "billing-documents-v1",
	})
	if err != nil {
		t.Fatalf("insert valid document: %v", err)
	}

	_, err = st.InsertBillingDocumentTx(ctx, tx, user.ID, BillingDocumentInput{
		InvoiceID:     invoice.ID,
		Kind:          "invoice_pdf",
		StoragePath:   "invoices/2026/MAIN/2026-0001/invoice-copy.pdf",
		SHA256:        validHash,
		ByteSize:      1024,
		RenderVersion: "billing-documents-v1",
	})
	if err == nil {
		t.Fatal("expected duplicate kind error")
	}

	_, err = st.InsertBillingDocumentTx(ctx, tx, user.ID, BillingDocumentInput{
		InvoiceID:     invoice.ID,
		Kind:          "work_protocol_pdf",
		StoragePath:   "../leotime.db",
		SHA256:        validHash,
		ByteSize:      1024,
		RenderVersion: "billing-documents-v1",
	})
	if err == nil {
		t.Fatal("expected path traversal error")
	}

	_, err = st.InsertBillingDocumentTx(ctx, tx, user.ID, BillingDocumentInput{
		InvoiceID:     invoice.ID,
		Kind:          "work_protocol_pdf",
		StoragePath:   "invoices/2026/MAIN/2026-0001/work-protocol.pdf",
		SHA256:        "not-a-valid-hash",
		ByteSize:      1024,
		RenderVersion: "billing-documents-v1",
	})
	if err == nil {
		t.Fatal("expected invalid hash error")
	}
}

func TestCreateInvoiceDraftPersistsPeriodFields(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)
	client := invoiceClient(t, ctx, st, user.ID)

	invoice := createTestInvoiceDraft(t, ctx, st, user.ID, InvoiceDraftFromTimeInput{
		ClientID:           client.ID,
		From:               "2026-07-01T00:00:00Z",
		To:                 "2026-07-31T23:59:59Z",
		PeriodFrom:         "2026-07-01T00:00:00Z",
		PeriodTo:           "2026-07-31T23:59:59Z",
		WorkProtocolDetail: "detailed",
	})

	if invoice.PeriodFrom != "2026-07-01T00:00:00Z" || invoice.PeriodTo != "2026-07-31T23:59:59Z" {
		t.Fatalf("unexpected period fields: %+v", invoice)
	}
	if invoice.WorkProtocolDetail != "detailed" {
		t.Fatalf("expected detailed work protocol, got %q", invoice.WorkProtocolDetail)
	}
	if !strings.HasPrefix(invoice.InvoiceNumber, "DRAFT-") {
		t.Fatalf("expected draft invoice number, got %q", invoice.InvoiceNumber)
	}
}

func createTestInvoiceDraft(t *testing.T, ctx context.Context, st *Store, userID string, input ...InvoiceDraftFromTimeInput) *Invoice {
	t.Helper()

	draftInput := InvoiceDraftFromTimeInput{
		From: "2026-07-01T00:00:00Z",
		To:   "2026-07-31T23:59:59Z",
	}
	if len(input) > 0 {
		draftInput = input[0]
	}
	if draftInput.ClientID == "" {
		draftInput.ClientID = invoiceClient(t, ctx, st, userID).ID
	}

	if draftInput.From == "" {
		draftInput.From = "2026-07-01T00:00:00Z"
	}
	if draftInput.To == "" {
		draftInput.To = "2026-07-31T23:59:59Z"
	}

	client, err := st.ClientByID(ctx, userID, draftInput.ClientID)
	if err != nil {
		t.Fatalf("load client: %v", err)
	}

	project, err := st.CreateProject(ctx, userID, ProjectInput{
		ClientID: client.ID,
		Name:     "Billing project",
		Color:    "#2563eb",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, userID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "Billable work",
		StartedAt:   "2026-07-01T08:00:00Z",
		EndedAt:     "2026-07-01T10:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	invoice, err := st.CreateInvoiceDraftFromTime(ctx, userID, draftInput)
	if err != nil {
		t.Fatalf("create invoice draft: %v", err)
	}
	return invoice
}

func invoiceClient(t *testing.T, ctx context.Context, st *Store, userID string) *Client {
	t.Helper()
	client, err := st.CreateClient(ctx, userID, ClientInput{
		Name:                   "Billing Client",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 10000,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return client
}
