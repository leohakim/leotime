package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/billing"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type billingDocumentResponse struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	SHA256      string `json:"sha256"`
	ByteSize    int64  `json:"byteSize"`
	DownloadURL string `json:"downloadUrl"`
}

func (s *Server) previewInvoice(w http.ResponseWriter, r *http.Request, user *store.User) {
	invoiceID := chi.URLParam(r, "invoiceID")
	invoice, err := s.store.InvoiceByID(r.Context(), user.ID, invoiceID)
	if err != nil {
		writeInvoiceError(w, err)
		return
	}
	if invoice.Status != "draft" {
		writeError(w, http.StatusConflict, "invoice_not_editable", "only draft invoices can be previewed")
		return
	}

	entries, err := s.store.TimeEntriesForInvoice(r.Context(), user.ID, invoice)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invoice_preview_failed", "load invoice entries failed")
		return
	}

	options, err := s.invoiceSnapshotOptions(r.Context(), user.ID, invoice, billing.SnapshotOptions{
		Preview: true,
		IssueAt: time.Now().UTC(),
	})
	if err != nil {
		writeInvoiceError(w, err)
		return
	}

	snapshot, err := billing.BuildDocumentSnapshot(invoice, entries, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invoice_preview_failed", "build invoice preview failed")
		return
	}

	html, err := s.renderer.RenderPreviewHTML(r.Context(), snapshot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invoice_preview_failed", "render invoice preview failed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(html)
}

func (s *Server) issueInvoice(w http.ResponseWriter, r *http.Request, user *store.User) {
	invoiceID := chi.URLParam(r, "invoiceID")
	invoice, err := s.store.InvoiceByID(r.Context(), user.ID, invoiceID)
	if err != nil {
		writeInvoiceError(w, err)
		return
	}
	if invoice.Status != "draft" {
		writeError(w, http.StatusConflict, "invoice_not_editable", "invoice is not editable")
		return
	}

	if strings.TrimSpace(invoice.SeriesID) == "" {
		defaultSeries, err := s.store.DefaultInvoiceSeries(r.Context(), user.ID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invoice_series_required", "fiscal series is required")
			return
		}
		seriesID := defaultSeries.ID
		if _, err = s.store.UpdateInvoice(r.Context(), user.ID, invoiceID, store.InvoiceUpdateInput{
			SeriesID: &seriesID,
		}); err != nil {
			writeInvoiceError(w, err)
			return
		}
	}

	issued, err := s.issueService.Issue(r.Context(), user.ID, billing.IssueRequest{
		InvoiceID: invoiceID,
		IssueAt:   time.Now().UTC(),
	})
	if err != nil {
		writeInvoiceIssueError(w, err)
		return
	}

	attachDocumentDownloadURLs(issued)
	writeJSON(w, http.StatusOK, issued)
}

func (s *Server) cancelInvoice(w http.ResponseWriter, r *http.Request, user *store.User) {
	var request struct {
		Reason string `json:"reason"`
	}
	if !decodeJSONBody(w, r, &request) {
		return
	}

	invoiceID := chi.URLParam(r, "invoiceID")
	invoice, err := s.store.CancelInvoice(r.Context(), user.ID, invoiceID, request.Reason)
	if err != nil {
		writeInvoiceError(w, err)
		return
	}
	attachDocumentDownloadURLs(invoice)
	writeJSON(w, http.StatusOK, invoice)
}

func (s *Server) listInvoiceDocuments(w http.ResponseWriter, r *http.Request, user *store.User) {
	invoiceID := chi.URLParam(r, "invoiceID")
	if _, err := s.store.InvoiceByID(r.Context(), user.ID, invoiceID); err != nil {
		writeInvoiceError(w, err)
		return
	}

	documents, err := s.store.ListInvoiceDocuments(r.Context(), user.ID, invoiceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invoice_documents_load_failed", "load invoice documents failed")
		return
	}

	payload := make([]billingDocumentResponse, 0, len(documents))
	for _, document := range documents {
		payload = append(payload, billingDocumentResponse{
			ID:          document.ID,
			Kind:        document.Kind,
			SHA256:      document.SHA256,
			ByteSize:    document.ByteSize,
			DownloadURL: invoiceDocumentDownloadURL(invoiceID, document.ID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": payload})
}

func (s *Server) downloadInvoiceDocument(w http.ResponseWriter, r *http.Request, user *store.User) {
	invoiceID := chi.URLParam(r, "invoiceID")
	documentID := chi.URLParam(r, "documentID")

	invoice, err := s.store.InvoiceByID(r.Context(), user.ID, invoiceID)
	if err != nil {
		writeInvoiceError(w, err)
		return
	}

	document, err := s.store.BillingDocumentByID(r.Context(), user.ID, invoiceID, documentID)
	if err != nil {
		if errors.Is(err, store.ErrBillingDocumentNotFound) {
			writeError(w, http.StatusNotFound, "invoice_document_not_found", "invoice document not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "invoice_document_download_failed", "load invoice document failed")
		return
	}

	file, _, err := s.documents.Open(document.StoragePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invoice_document_download_failed", "open invoice document failed")
		return
	}
	defer file.Close()

	filename := billing.SafeDownloadFilename(invoice.InvoiceNumber, documentFilenameSuffix(document.Kind))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", billing.ContentDispositionAttachment(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

func (s *Server) invoiceSnapshotOptions(ctx context.Context, userID string, invoice *store.Invoice, options billing.SnapshotOptions) (billing.SnapshotOptions, error) {
	if strings.TrimSpace(invoice.SeriesID) == "" {
		return options, nil
	}
	series, err := s.store.InvoiceSeriesByID(ctx, userID, invoice.SeriesID)
	if err != nil {
		return options, err
	}
	options.SeriesCode = series.Code
	return options, nil
}

func attachDocumentDownloadURLs(invoice *store.Invoice) {
	if invoice == nil {
		return
	}
	for index := range invoice.Documents {
		invoice.Documents[index].DownloadURL = invoiceDocumentDownloadURL(invoice.ID, invoice.Documents[index].ID)
	}
}

func invoiceDocumentDownloadURL(invoiceID, documentID string) string {
	return fmt.Sprintf("/api/v1/invoices/%s/documents/%s/download", invoiceID, documentID)
}

func documentFilenameSuffix(kind string) string {
	switch kind {
	case "work_protocol_pdf":
		return "-work-protocol.pdf"
	default:
		return "-invoice.pdf"
	}
}

func writeInvoiceIssueError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidInvoiceInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrInvoiceNotFound):
		writeError(w, http.StatusNotFound, "invoice_not_found", "invoice not found")
	case errors.Is(err, store.ErrInvoiceNotEditable):
		writeError(w, http.StatusConflict, "invoice_not_editable", "invoice is not editable")
	case errors.Is(err, store.ErrInvoiceSeriesNotFound):
		writeError(w, http.StatusBadRequest, "invoice_series_required", "fiscal series is required")
	default:
		writeError(w, http.StatusInternalServerError, "invoice_issue_failed", "invoice issue failed")
	}
}
