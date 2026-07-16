package store

import (
	"context"
	"testing"
)

func TestClearUserDataRemovesProductRows(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Reset Client", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	project, err := st.CreateProject(ctx, user.ID, ProjectInput{ClientID: client.ID, Name: "Reset Project"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	tag, err := st.CreateTag(ctx, user.ID, TagInput{Name: "Reset tag"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if _, err := st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "Reset me",
		StartedAt:   "2026-07-01T09:00:00Z",
		EndedAt:     "2026-07-01T10:00:00Z",
		TagIDs:      []string{tag.ID},
	}); err != nil {
		t.Fatalf("create time entry: %v", err)
	}

	if _, err := st.DB().ExecContext(ctx, `
		INSERT INTO external_mappings (id, provider, external_type, external_id, internal_type, internal_id, created_at, updated_at)
		VALUES ('map_reset', 'solidtime', 'client', 'ext-1', 'client', ?, ?, ?)
	`, client.ID, nowString(), nowString()); err != nil {
		t.Fatalf("insert external mapping: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx, `
		INSERT INTO import_runs (id, provider, source_path, dry_run, status, summary_json, started_at)
		VALUES ('run_reset', 'solidtime', '/tmp/export.zip', 0, 'completed', '{}', ?)
	`, nowString()); err != nil {
		t.Fatalf("insert import run: %v", err)
	}

	summary, err := st.ClearUserData(ctx, user.ID)
	if err != nil {
		t.Fatalf("clear user data: %v", err)
	}
	if summary.Clients != 1 || summary.Projects != 1 || summary.Tags != 1 || summary.TimeEntries != 1 {
		t.Fatalf("unexpected reset summary: %+v", summary)
	}
	if summary.ExternalMappings != 1 || summary.ImportRuns != 1 {
		t.Fatalf("expected import artifacts cleared, got %+v", summary)
	}

	overview, err := st.Overview(ctx, user.ID)
	if err != nil {
		t.Fatalf("overview after reset: %v", err)
	}
	if overview.ClientsTotal != 0 || overview.TimeEntriesTotal != 0 || overview.TagsTotal != 0 {
		t.Fatalf("expected empty product data, got %+v", overview)
	}

	series, err := st.ListInvoiceSeries(ctx, user.ID)
	if err != nil {
		t.Fatalf("list invoice series after reset: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected default invoice series to be recreated, got %d", len(series))
	}
}

func TestClearUserDataRemovesBillingDocumentsMetadata(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	invoiceID, err := newID("inv")
	if err != nil {
		t.Fatalf("new invoice id: %v", err)
	}
	now := nowString()
	if _, err := st.DB().ExecContext(ctx, `
		INSERT INTO invoices (
			id, user_id, invoice_number, status, currency, seller_name, client_name,
			subtotal_minor, tax_minor, withholding_minor, total_minor, created_at, updated_at
		) VALUES (?, ?, '2026-0001', 'draft', 'EUR', 'Seller', 'Client', 0, 0, 0, 0, ?, ?)
	`, invoiceID, user.ID, now, now); err != nil {
		t.Fatalf("insert invoice: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx, `
		INSERT INTO billing_documents (
			id, user_id, invoice_id, kind, storage_path, sha256, byte_size, mime_type, render_version, created_at
		) VALUES ('doc_reset', ?, ?, 'invoice_pdf', 'invoices/2026/seed.pdf', ?, 10, 'application/pdf', 'v1', ?)
	`, user.ID, invoiceID, repeat("a", 64), now); err != nil {
		t.Fatalf("insert billing document: %v", err)
	}

	summary, err := st.ClearUserData(ctx, user.ID)
	if err != nil {
		t.Fatalf("clear user data: %v", err)
	}
	if summary.BillingDocuments != 1 || len(summary.DocumentPaths) != 1 {
		t.Fatalf("expected billing document metadata removed, got %+v", summary)
	}
}

func TestClearUserDataPreservesUserAccount(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	if _, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Keep account"}); err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := st.ClearUserData(ctx, user.ID); err != nil {
		t.Fatalf("clear user data: %v", err)
	}

	loaded, err := st.UserByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("load user after reset: %v", err)
	}
	if loaded.ID != user.ID || loaded.Email != user.Email {
		t.Fatalf("expected user account to remain, got %+v", loaded)
	}
}

func repeat(value string, count int) string {
	out := make([]byte, count)
	for i := range out {
		out[i] = value[0]
	}
	return string(out)
}
