package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/backup"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func (s *Server) getBackupSettings(w http.ResponseWriter, r *http.Request, user *store.User) {
	settings, err := s.backups.GetSettings(r.Context(), user.ID)
	if err != nil {
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) putBackupSettings(w http.ResponseWriter, r *http.Request, user *store.User) {
	var input store.BackupSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	settings, err := s.backups.SaveSettings(r.Context(), user.ID, input)
	if err != nil {
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) testBackupConnection(w http.ResponseWriter, r *http.Request, user *store.User) {
	var draft *store.BackupSettingsInput
	var body store.BackupSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil && backupDraftProvided(body) {
		draft = &body
	}

	if err := s.backups.TestConnection(r.Context(), user.ID, draft); err != nil {
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "connection_ok",
	})
}

func backupDraftProvided(body store.BackupSettingsInput) bool {
	return strings.TrimSpace(body.Bucket) != "" ||
		strings.TrimSpace(body.AccessKeyID) != "" ||
		strings.TrimSpace(body.Endpoint) != "" ||
		strings.TrimSpace(body.SecretAccessKey) != ""
}

func (s *Server) runBackup(w http.ResponseWriter, r *http.Request, user *store.User) {
	result, err := s.backups.Run(r.Context(), user.ID, true)
	if err != nil {
		if errors.Is(err, backup.ErrBusy) {
			writeError(w, http.StatusConflict, "backup_busy", "backup already running")
			return
		}
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listBackupObjects(w http.ResponseWriter, r *http.Request, user *store.User) {
	objects, err := s.backups.ListObjects(r.Context(), user.ID)
	if err != nil {
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"objects": objects})
}

func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request, user *store.User) {
	var request struct {
		ObjectKey string `json:"objectKey"`
		Latest    bool   `json:"latest"`
		Confirm   bool   `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if !request.Confirm {
		writeError(w, http.StatusBadRequest, "confirm_required", "confirm is required")
		return
	}

	result, err := s.backups.Restore(r.Context(), user.ID, request.ObjectKey, request.Latest)
	if err != nil {
		if errors.Is(err, backup.ErrBusy) {
			writeError(w, http.StatusConflict, "backup_busy", "backup job already running")
			return
		}
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getBackupStatus(w http.ResponseWriter, r *http.Request, user *store.User) {
	settings, err := s.backups.GetSettings(r.Context(), user.ID)
	if err != nil {
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lastRunAt":            settings.LastRunAt,
		"lastStatus":           settings.LastStatus,
		"lastError":            settings.LastError,
		"lastObjectKey":        settings.LastObjectKey,
		"lastRestoreAt":        settings.LastRestoreAt,
		"lastRestoreStatus":    settings.LastRestoreStatus,
		"lastRestoreError":     settings.LastRestoreError,
		"lastRestoreObjectKey": settings.LastRestoreObjectKey,
	})
}

func writeBackupError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrBackupSecretsKeyMissing):
		writeError(w, http.StatusServiceUnavailable, "backup_secrets_key_missing", "backup secrets key is not configured")
	case store.IsValidation(err, store.ErrInvalidBackupSettings):
		writeValidationStoreError(w, err)
	default:
		message := strings.TrimSpace(err.Error())
		if message == "" {
			message = "backup operation failed"
		}
		writeError(w, http.StatusBadGateway, "backup_operation_failed", message)
	}
}
