package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/leotime/leotime/apps/api/internal/store"
)

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.passwordReset.RequestReset(r.Context(), request.Email); err != nil {
		writeError(w, http.StatusInternalServerError, "password reset request failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.passwordReset.ResetPassword(r.Context(), request.Token, request.NewPassword); err != nil {
		if errors.Is(err, store.ErrInvalidPasswordReset) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "password reset failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
