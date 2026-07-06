package httpapi

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/solidtimeimport"
	"github.com/leotime/leotime/apps/api/internal/store"
)

const maxSolidtimeImportBytes = 32 << 20

type solidtimeImportResponse struct {
	Summary solidtimeimport.Summary `json:"summary"`
}

func (s *Server) importSolidtime(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := r.ParseMultipartForm(maxSolidtimeImportBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	if header.Size > maxSolidtimeImportBytes {
		writeError(w, http.StatusBadRequest, "file exceeds 32MB limit")
		return
	}

	filename := strings.ToLower(strings.TrimSpace(header.Filename))
	if filename != "" && !strings.HasSuffix(filename, ".zip") {
		writeError(w, http.StatusBadRequest, "file must be a .zip export")
		return
	}

	tempFile, err := os.CreateTemp("", "leotime-solidtime-*.zip")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create temp file failed")
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	defer tempFile.Close()

	written, err := io.Copy(tempFile, io.LimitReader(file, maxSolidtimeImportBytes+1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read upload failed")
		return
	}
	if written > maxSolidtimeImportBytes {
		writeError(w, http.StatusBadRequest, "file exceeds 32MB limit")
		return
	}
	if err := tempFile.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "save upload failed")
		return
	}

	dryRun := parseBoolQuery(r, "dryRun") || strings.EqualFold(strings.TrimSpace(r.FormValue("dryRun")), "true")

	importer := solidtimeimport.New(s.store.DB())
	summary, err := importer.ImportFile(r.Context(), solidtimeimport.Options{
		FilePath:  tempPath,
		UserEmail: user.Email,
		DryRun:    dryRun,
	})
	if err != nil {
		status := http.StatusBadRequest
		if summary.ExportID != "" || summary.Provider != "" || len(summary.Errors) > 0 {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, solidtimeImportResponse{Summary: summary})
		return
	}

	writeJSON(w, http.StatusOK, solidtimeImportResponse{Summary: summary})
}

func parseBoolQuery(r *http.Request, key string) bool {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	return strings.EqualFold(value, "true") || value == "1"
}
