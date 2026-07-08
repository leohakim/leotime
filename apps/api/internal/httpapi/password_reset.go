package httpapi

import (
	"errors"
	"net/http"

	"github.com/leotime/leotime/apps/api/internal/store"
)

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email string `json:"email"`
	}
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if !s.rateLimitForgotPassword(w, r, request.Email) {
		return
	}

	if err := s.passwordReset.RequestReset(r.Context(), request.Email); err != nil {
		writeError(w, http.StatusInternalServerError, "password_reset_request_failed", "password reset request failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if !decodeJSONBody(w, r, &request) {
		return
	}

	if err := s.passwordReset.ResetPassword(r.Context(), request.Token, request.NewPassword); err != nil {
		if errors.Is(err, store.ErrInvalidPasswordReset) || store.IsValidation(err, store.ErrInvalidPasswordReset) {
			writeValidationStoreError(w, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "password_reset_failed", "password reset failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
