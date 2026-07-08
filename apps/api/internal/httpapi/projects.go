package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type projectsResponse struct {
	Projects []store.Project `json:"projects"`
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request, user *store.User) {
	includeArchived := r.URL.Query().Get("includeArchived") == "true"
	clientID := r.URL.Query().Get("clientId")
	projects, err := s.store.ListProjects(r.Context(), user.ID, includeArchived, clientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "projects_load_failed", "load projects failed")
		return
	}
	writeJSON(w, http.StatusOK, projectsResponse{Projects: projects})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeProjectInput(w, r)
	if !ok {
		return
	}

	project, err := s.store.CreateProject(r.Context(), user.ID, input)
	if err != nil {
		writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request, user *store.User) {
	project, err := s.store.ProjectByID(r.Context(), user.ID, chi.URLParam(r, "projectID"))
	if err != nil {
		writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeProjectInput(w, r)
	if !ok {
		return
	}

	project, err := s.store.UpdateProject(r.Context(), user.ID, chi.URLParam(r, "projectID"), input)
	if err != nil {
		writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) archiveProject(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.ArchiveProject(r.Context(), user.ID, chi.URLParam(r, "projectID")); err != nil {
		writeProjectError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreProject(w http.ResponseWriter, r *http.Request, user *store.User) {
	project, err := s.store.RestoreProject(r.Context(), user.ID, chi.URLParam(r, "projectID"))
	if err != nil {
		writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func decodeProjectInput(w http.ResponseWriter, r *http.Request) (store.ProjectInput, bool) {
	var input store.ProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return store.ProjectInput{}, false
	}
	return input, true
}

func writeProjectError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidProjectInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrProjectNotFound):
		writeError(w, http.StatusNotFound, "project_not_found", "project not found")
	default:
		writeError(w, http.StatusInternalServerError, "project_operation_failed", "project operation failed")
	}
}
