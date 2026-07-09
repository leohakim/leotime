package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type invoiceSeriesResponse struct {
	Series []store.InvoiceSeries `json:"series"`
}

func (s *Server) listInvoiceSeries(w http.ResponseWriter, r *http.Request, user *store.User) {
	series, err := s.store.ListInvoiceSeries(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invoice_series_load_failed", "load invoice series failed")
		return
	}
	writeJSON(w, http.StatusOK, invoiceSeriesResponse{Series: series})
}

func (s *Server) createInvoiceSeries(w http.ResponseWriter, r *http.Request, user *store.User) {
	var input store.InvoiceSeriesInput
	if !decodeJSONBody(w, r, &input) {
		return
	}

	series, err := s.store.CreateInvoiceSeries(r.Context(), user.ID, input)
	if err != nil {
		writeInvoiceSeriesError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, series)
}

func (s *Server) updateInvoiceSeries(w http.ResponseWriter, r *http.Request, user *store.User) {
	var input store.InvoiceSeriesInput
	if !decodeJSONBody(w, r, &input) {
		return
	}

	series, err := s.store.UpdateInvoiceSeries(r.Context(), user.ID, chi.URLParam(r, "seriesID"), input)
	if err != nil {
		writeInvoiceSeriesError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, series)
}

func writeInvoiceSeriesError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidInvoiceSeriesInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrInvoiceSeriesNotFound):
		writeError(w, http.StatusNotFound, "invoice_series_not_found", "invoice series not found")
	default:
		writeError(w, http.StatusInternalServerError, "invoice_series_operation_failed", "invoice series operation failed")
	}
}
