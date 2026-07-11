package httpapi

import (
	"bytes"
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/billing"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type invoicesResponse struct {
	Invoices []store.Invoice `json:"invoices"`
}

func (s *Server) listInvoices(w http.ResponseWriter, r *http.Request, user *store.User) {
	invoices, err := s.store.ListInvoices(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invoices_load_failed", "load invoices failed")
		return
	}
	writeJSON(w, http.StatusOK, invoicesResponse{Invoices: invoices})
}

func (s *Server) createInvoiceDraftFromTime(w http.ResponseWriter, r *http.Request, user *store.User) {
	var input store.InvoiceDraftFromTimeInput
	if !decodeJSONBody(w, r, &input) {
		return
	}

	invoice, err := s.store.CreateInvoiceDraftFromTime(r.Context(), user.ID, input)
	if err != nil {
		writeInvoiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, invoice)
}

func (s *Server) getInvoice(w http.ResponseWriter, r *http.Request, user *store.User) {
	invoice, err := s.store.InvoiceByID(r.Context(), user.ID, chi.URLParam(r, "invoiceID"))
	if err != nil {
		writeInvoiceError(w, err)
		return
	}
	attachDocumentDownloadURLs(invoice)
	writeJSON(w, http.StatusOK, invoice)
}

func (s *Server) updateInvoice(w http.ResponseWriter, r *http.Request, user *store.User) {
	var input store.InvoiceUpdateInput
	if !decodeJSONBody(w, r, &input) {
		return
	}

	invoice, err := s.store.UpdateInvoice(r.Context(), user.ID, chi.URLParam(r, "invoiceID"), input)
	if err != nil {
		writeInvoiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invoice)
}

func (s *Server) updateInvoiceStatus(w http.ResponseWriter, r *http.Request, user *store.User) {
	var request struct {
		Status string `json:"status"`
	}
	if !decodeJSONBody(w, r, &request) {
		return
	}

	invoice, err := s.store.UpdateInvoiceStatus(r.Context(), user.ID, chi.URLParam(r, "invoiceID"), request.Status)
	if err != nil {
		writeInvoiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invoice)
}

func (s *Server) deleteInvoice(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.DeleteInvoice(r.Context(), user.ID, chi.URLParam(r, "invoiceID")); err != nil {
		writeInvoiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) exportInvoice(w http.ResponseWriter, r *http.Request, user *store.User) {
	invoice, err := s.store.InvoiceByID(r.Context(), user.ID, chi.URLParam(r, "invoiceID"))
	if err != nil {
		writeInvoiceError(w, err)
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "html"
	}

	switch format {
	case "html":
		payload := s.store.RenderInvoiceHTML(invoice)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", billing.ContentDispositionAttachment(billing.SafeDownloadFilename(invoice.InvoiceNumber, ".html")))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	case "json":
		w.Header().Set("Content-Disposition", billing.ContentDispositionAttachment(billing.SafeDownloadFilename(invoice.InvoiceNumber, ".json")))
		writeJSON(w, http.StatusOK, invoice)
	case "csv":
		payload, err := renderInvoiceCSV(invoice)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "report_render_failed", "render csv failed")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", billing.ContentDispositionAttachment(billing.SafeDownloadFilename(invoice.InvoiceNumber, ".csv")))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	default:
		writeError(w, http.StatusBadRequest, "invalid_format", "format must be html, csv, or json")
	}
}

func renderInvoiceCSV(invoice *store.Invoice) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	if err := writer.Write([]string{"invoice_number", "status", "currency", "client", "subtotal_minor", "tax_minor", "withholding_minor", "total_minor"}); err != nil {
		return nil, err
	}
	if err := writer.Write([]string{
		invoice.InvoiceNumber,
		invoice.Status,
		invoice.Currency,
		invoice.ClientName,
		strconv.FormatInt(invoice.SubtotalMinor, 10),
		strconv.FormatInt(invoice.TaxMinor, 10),
		strconv.FormatInt(invoice.WithholdingMinor, 10),
		strconv.FormatInt(invoice.TotalMinor, 10),
	}); err != nil {
		return nil, err
	}
	if err := writer.Write([]string{"description", "quantity_minutes", "unit_rate_minor", "subtotal_minor", "tax_rate_basis_points"}); err != nil {
		return nil, err
	}
	for _, line := range invoice.Lines {
		if err := writer.Write([]string{
			line.Description,
			strconv.Itoa(line.QuantityMinutes),
			strconv.FormatInt(line.UnitRateMinor, 10),
			strconv.FormatInt(line.SubtotalMinor, 10),
			strconv.Itoa(line.TaxRateBasisPoints),
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

func writeInvoiceError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidInvoiceInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrInvoiceNotFound):
		writeError(w, http.StatusNotFound, "invoice_not_found", "invoice not found")
	case errors.Is(err, store.ErrInvoiceNotEditable):
		writeError(w, http.StatusConflict, "invoice_not_editable", "invoice is not editable")
	default:
		writeError(w, http.StatusInternalServerError, "invoice_operation_failed", "invoice operation failed")
	}
}
