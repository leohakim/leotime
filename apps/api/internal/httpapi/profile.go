package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/leotime/leotime/apps/api/internal/store"
)

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request, user *store.User) {
	profile, err := s.store.ProfileByUserID(r.Context(), user.ID)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, user *store.User) {
	var input store.ProfileUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	profile, err := s.store.UpdateProfile(r.Context(), user.ID, input)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request, user *store.User) {
	var input store.ChangePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	if err := s.store.ChangePassword(r.Context(), user.ID, input); err != nil {
		writeProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "profile not found")
	case store.IsValidation(err, store.ErrInvalidProfileInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "email already in use")
	case store.IsValidation(err, store.ErrInvalidPasswordChange):
		writeValidationStoreError(w, err)
	default:
		writeError(w, http.StatusInternalServerError, "profile_operation_failed", "profile operation failed")
	}
}
