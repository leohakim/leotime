package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type timeEntriesResponse struct {
	TimeEntries []store.TimeEntry `json:"timeEntries"`
}

func (s *Server) listTimeEntries(w http.ResponseWriter, r *http.Request, user *store.User) {
	options := store.TimeEntryListOptions{
		From:      r.URL.Query().Get("from"),
		To:        r.URL.Query().Get("to"),
		ClientID:  r.URL.Query().Get("clientId"),
		ProjectID: r.URL.Query().Get("projectId"),
		TaskID:    r.URL.Query().Get("taskId"),
	}

	entries, err := s.store.ListTimeEntries(r.Context(), user.ID, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "time_entries_load_failed", "load time entries failed")
		return
	}
	writeJSON(w, http.StatusOK, timeEntriesResponse{TimeEntries: entries})
}

func (s *Server) createTimeEntry(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeTimeEntryInput(w, r)
	if !ok {
		return
	}

	entry, err := s.store.CreateTimeEntry(r.Context(), user.ID, input)
	if err != nil {
		writeTimeEntryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) getTimeEntry(w http.ResponseWriter, r *http.Request, user *store.User) {
	entry, err := s.store.TimeEntryByID(r.Context(), user.ID, chi.URLParam(r, "timeEntryID"))
	if err != nil {
		writeTimeEntryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) updateTimeEntry(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeTimeEntryInput(w, r)
	if !ok {
		return
	}

	entry, err := s.store.UpdateTimeEntry(r.Context(), user.ID, chi.URLParam(r, "timeEntryID"), input)
	if err != nil {
		writeTimeEntryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) deleteTimeEntry(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.DeleteTimeEntry(r.Context(), user.ID, chi.URLParam(r, "timeEntryID")); err != nil {
		writeTimeEntryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeTimeEntryInput(w http.ResponseWriter, r *http.Request) (store.TimeEntryInput, bool) {
	var input store.TimeEntryInput
	if !decodeJSONBody(w, r, &input) {
		return store.TimeEntryInput{}, false
	}
	return input, true
}

func writeTimeEntryError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidTimeEntryInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrTimeEntryNotFound):
		writeError(w, http.StatusNotFound, "time_entry_not_found", "time entry not found")
	default:
		writeError(w, http.StatusInternalServerError, "time_entry_operation_failed", "time entry operation failed")
	}
}
