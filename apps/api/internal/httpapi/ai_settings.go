package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/backup/crypto"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type aiSettingsUpdateRequest struct {
	Enabled        bool   `json:"enabled"`
	GitAuthorEmail string `json:"gitAuthorEmail"`
	CursorAPIKey   string `json:"cursorApiKey"`
}

func (s *Server) getAISettings(w http.ResponseWriter, r *http.Request, user *store.User) {
	settings, err := s.store.AISettingsByUserID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ai_settings_load_failed", "load ai settings failed")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) putAISettings(w http.ResponseWriter, r *http.Request, user *store.User) {
	var body aiSettingsUpdateRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}

	cursorKeyEnc := ""
	if strings.TrimSpace(body.CursorAPIKey) != "" {
		encoded, err := s.encryptSecret(strings.TrimSpace(body.CursorAPIKey))
		if err != nil {
			writeAISettingsError(w, err)
			return
		}
		cursorKeyEnc = encoded
	}

	settings, err := s.store.UpsertAISettings(r.Context(), user.ID, store.AISettingsInput{
		Enabled:        body.Enabled,
		GitAuthorEmail: body.GitAuthorEmail,
	}, cursorKeyEnc)
	if err != nil {
		writeAISettingsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) encryptSecret(plaintext string) (string, error) {
	if strings.TrimSpace(s.cfg.SecretsKey) == "" {
		return "", store.ErrAISecretsKeyMissing
	}
	key, err := crypto.ParseKey(s.cfg.SecretsKey)
	if err != nil {
		return "", err
	}
	return crypto.Encrypt([]byte(plaintext), key)
}

func writeAISettingsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrAISecretsKeyMissing):
		writeError(w, http.StatusBadRequest, "secrets_key_missing", "LEOTIME_SECRETS_KEY is required to store secrets")
	case store.IsValidation(err, store.ErrInvalidProfileInput):
		writeValidationStoreError(w, err)
	default:
		writeError(w, http.StatusInternalServerError, "ai_settings_save_failed", "save ai settings failed")
	}
}
