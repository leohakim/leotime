package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"strings"
)

var ErrBillingDocumentNotFound = errors.New("billing document not found")

type BillingDocument struct {
	ID            string `json:"id"`
	InvoiceID     string `json:"invoiceId"`
	Kind          string `json:"kind"`
	StoragePath   string `json:"storagePath"`
	SHA256        string `json:"sha256"`
	ByteSize      int64  `json:"byteSize"`
	MimeType      string `json:"mimeType"`
	RenderVersion string `json:"renderVersion"`
	CreatedAt     string `json:"createdAt"`
}

type BillingDocumentInput struct {
	InvoiceID     string
	Kind          string
	StoragePath   string
	SHA256        string
	ByteSize      int64
	MimeType      string
	RenderVersion string
}

func (s *Store) ListInvoiceDocuments(ctx context.Context, userID, invoiceID string) ([]BillingDocument, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, invoice_id, kind, storage_path, sha256, byte_size, mime_type, render_version, created_at
		FROM billing_documents
		WHERE user_id = ? AND invoice_id = ?
		ORDER BY created_at ASC, id ASC
	`, userID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("list billing documents: %w", err)
	}
	defer rows.Close()

	var documents []BillingDocument
	for rows.Next() {
		doc, err := scanBillingDocument(rows)
		if err != nil {
			return nil, err
		}
		documents = append(documents, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate billing documents: %w", err)
	}
	return documents, nil
}

func (s *Store) BillingDocumentByID(ctx context.Context, userID, invoiceID, documentID string) (*BillingDocument, error) {
	doc, err := queryBillingDocument(ctx, s.db, `
		SELECT id, invoice_id, kind, storage_path, sha256, byte_size, mime_type, render_version, created_at
		FROM billing_documents
		WHERE user_id = ? AND invoice_id = ? AND id = ?
	`, userID, invoiceID, documentID)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Store) InsertBillingDocumentTx(ctx context.Context, tx *sql.Tx, userID string, input BillingDocumentInput) (*BillingDocument, error) {
	normalized, err := normalizeBillingDocumentInput(input)
	if err != nil {
		return nil, err
	}

	docID, err := newID("doc")
	if err != nil {
		return nil, err
	}
	now := nowString()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO billing_documents (
			id, user_id, invoice_id, kind, storage_path, sha256, byte_size, mime_type, render_version, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, docID, userID, normalized.InvoiceID, normalized.Kind, normalized.StoragePath,
		normalized.SHA256, normalized.ByteSize, normalized.MimeType, normalized.RenderVersion, now)
	if err != nil {
		return nil, fmt.Errorf("insert billing document: %w", err)
	}

	return queryBillingDocument(ctx, tx, `
		SELECT id, invoice_id, kind, storage_path, sha256, byte_size, mime_type, render_version, created_at
		FROM billing_documents
		WHERE user_id = ? AND id = ?
	`, userID, docID)
}

func (s *Store) CancelInvoice(ctx context.Context, userID, invoiceID, reason string) (*Invoice, error) {
	invoice, err := s.InvoiceByID(ctx, userID, invoiceID)
	if err != nil {
		return nil, err
	}
	switch invoice.Status {
	case "issued", "paid":
	default:
		return nil, validationError(ErrInvalidInvoiceInput, "status", "invalid", "only issued or paid invoices can be cancelled")
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, validationError(ErrInvalidInvoiceInput, "reason", "required", "cancellation reason is required")
	}

	now := nowString()
	_, err = s.db.ExecContext(ctx, `
		UPDATE invoices
		SET status = 'cancelled', cancelled_at = ?, cancellation_reason = ?, updated_at = ?
		WHERE user_id = ? AND id = ?
	`, now, reason, now, userID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("cancel invoice: %w", err)
	}
	return s.InvoiceByID(ctx, userID, invoiceID)
}

func normalizeBillingDocumentInput(input BillingDocumentInput) (BillingDocumentInput, error) {
	invoiceID := strings.TrimSpace(input.InvoiceID)
	if invoiceID == "" {
		return BillingDocumentInput{}, validationError(ErrInvalidInvoiceInput, "invoiceId", "required", "invoice is required")
	}

	kind := strings.TrimSpace(input.Kind)
	switch kind {
	case "invoice_pdf", "work_protocol_pdf":
	default:
		return BillingDocumentInput{}, validationError(ErrInvalidInvoiceInput, "kind", "invalid", "invalid document kind")
	}

	storagePath := strings.TrimSpace(strings.ReplaceAll(input.StoragePath, "\\", "/"))
	if storagePath == "" {
		return BillingDocumentInput{}, validationError(ErrInvalidInvoiceInput, "storagePath", "required", "storage path is required")
	}
	if strings.Contains(storagePath, "..") || path.IsAbs(storagePath) {
		return BillingDocumentInput{}, validationError(ErrInvalidInvoiceInput, "storagePath", "invalid", "storage path is invalid")
	}

	sha256 := strings.ToLower(strings.TrimSpace(input.SHA256))
	if !validSHA256(sha256) {
		return BillingDocumentInput{}, validationError(ErrInvalidInvoiceInput, "sha256", "invalid", "sha256 must be 64 lowercase hex characters")
	}

	if input.ByteSize <= 0 {
		return BillingDocumentInput{}, validationError(ErrInvalidInvoiceInput, "byteSize", "invalid", "byte size must be positive")
	}

	mimeType := strings.TrimSpace(input.MimeType)
	if mimeType == "" {
		mimeType = "application/pdf"
	}
	if mimeType != "application/pdf" {
		return BillingDocumentInput{}, validationError(ErrInvalidInvoiceInput, "mimeType", "invalid", "mime type must be application/pdf")
	}

	renderVersion := strings.TrimSpace(input.RenderVersion)
	if renderVersion == "" {
		return BillingDocumentInput{}, validationError(ErrInvalidInvoiceInput, "renderVersion", "required", "render version is required")
	}

	return BillingDocumentInput{
		InvoiceID:     invoiceID,
		Kind:          kind,
		StoragePath:   storagePath,
		SHA256:        sha256,
		ByteSize:      input.ByteSize,
		MimeType:      mimeType,
		RenderVersion: renderVersion,
	}, nil
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

type billingDocumentScanner interface {
	Scan(dest ...any) error
}

func scanBillingDocument(scanner billingDocumentScanner) (BillingDocument, error) {
	var doc BillingDocument
	if err := scanner.Scan(
		&doc.ID, &doc.InvoiceID, &doc.Kind, &doc.StoragePath, &doc.SHA256,
		&doc.ByteSize, &doc.MimeType, &doc.RenderVersion, &doc.CreatedAt,
	); err != nil {
		return BillingDocument{}, err
	}
	return doc, nil
}

func queryBillingDocument(ctx context.Context, db queryer, query string, args ...any) (*BillingDocument, error) {
	doc, err := scanBillingDocument(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBillingDocumentNotFound
		}
		return nil, fmt.Errorf("query billing document: %w", err)
	}
	return &doc, nil
}

type InvoiceIssueInput struct {
	InvoiceNumber        string
	SeriesID             string
	FiscalSequence       int
	IssuedAt             string
	DocumentSnapshotJSON string
}

func (s *Store) MarkInvoiceIssuedTx(ctx context.Context, tx *sql.Tx, userID, invoiceID string, input InvoiceIssueInput) error {
	now := nowString()
	result, err := tx.ExecContext(ctx, `
		UPDATE invoices
		SET status = 'issued', invoice_number = ?, series_id = ?, fiscal_sequence = ?,
			issued_at = ?, document_snapshot_json = ?, updated_at = ?
		WHERE user_id = ? AND id = ? AND status = 'draft'
	`, input.InvoiceNumber, input.SeriesID, input.FiscalSequence, input.IssuedAt,
		input.DocumentSnapshotJSON, now, userID, invoiceID)
	if err != nil {
		return fmt.Errorf("mark invoice issued: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark invoice issued rows: %w", err)
	}
	if rows == 0 {
		return ErrInvoiceNotEditable
	}
	return nil
}

func (s *Store) TimeEntriesForInvoice(ctx context.Context, userID string, invoice *Invoice) ([]TimeEntry, error) {
	ids := make([]string, 0, len(invoice.Lines))
	for _, line := range invoice.Lines {
		if strings.TrimSpace(line.TimeEntryID) != "" {
			ids = append(ids, line.TimeEntryID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	entries := make([]TimeEntry, 0, len(ids))
	for _, id := range ids {
		entry, err := s.TimeEntryByID(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, nil
}
