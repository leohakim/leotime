package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type timersResponse struct {
	Timers []store.TimeEntry `json:"timers"`
}

func (s *Server) listTimers(w http.ResponseWriter, r *http.Request, user *store.User) {
	timers, err := s.store.ListOpenTimers(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "timers_load_failed", "load timers failed")
		return
	}
	writeJSON(w, http.StatusOK, timersResponse{Timers: timers})
}

func (s *Server) startTimer(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeTimerStartInput(w, r)
	if !ok {
		return
	}

	timer, err := s.store.StartTimer(r.Context(), user.ID, input)
	if err != nil {
		writeTimerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, timer)
}

func (s *Server) updateTimer(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeTimerStartInput(w, r)
	if !ok {
		return
	}

	timer, err := s.store.UpdateOpenTimer(r.Context(), user.ID, chi.URLParam(r, "timeEntryID"), input)
	if err != nil {
		writeTimerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, timer)
}

func (s *Server) stopTimer(w http.ResponseWriter, r *http.Request, user *store.User) {
	timer, err := s.store.StopTimer(r.Context(), user.ID, chi.URLParam(r, "timeEntryID"))
	if err != nil {
		writeTimerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, timer)
}

func (s *Server) discardTimer(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.DiscardTimer(r.Context(), user.ID, chi.URLParam(r, "timeEntryID")); err != nil {
		writeTimerError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeTimerStartInput(w http.ResponseWriter, r *http.Request) (store.TimerStartInput, bool) {
	var input store.TimerStartInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return store.TimerStartInput{}, false
	}
	return input, true
}

func writeTimerError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidTimeEntryInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrTimerNotFound):
		writeError(w, http.StatusNotFound, "timer_not_found", "timer not found")
	default:
		writeError(w, http.StatusInternalServerError, "timer_operation_failed", "timer operation failed")
	}
}
