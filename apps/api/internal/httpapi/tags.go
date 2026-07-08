package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type tagsResponse struct {
	Tags []store.Tag `json:"tags"`
}

func (s *Server) listTags(w http.ResponseWriter, r *http.Request, user *store.User) {
	includeArchived := r.URL.Query().Get("includeArchived") == "true"
	tags, err := s.store.ListTags(r.Context(), user.ID, includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tags_load_failed", "load tags failed")
		return
	}
	writeJSON(w, http.StatusOK, tagsResponse{Tags: tags})
}

func (s *Server) createTag(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeTagInput(w, r)
	if !ok {
		return
	}

	tag, err := s.store.CreateTag(r.Context(), user.ID, input)
	if err != nil {
		writeTagError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func (s *Server) getTag(w http.ResponseWriter, r *http.Request, user *store.User) {
	tag, err := s.store.TagByID(r.Context(), user.ID, chi.URLParam(r, "tagID"))
	if err != nil {
		writeTagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func (s *Server) updateTag(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeTagInput(w, r)
	if !ok {
		return
	}

	tag, err := s.store.UpdateTag(r.Context(), user.ID, chi.URLParam(r, "tagID"), input)
	if err != nil {
		writeTagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func (s *Server) archiveTag(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.ArchiveTag(r.Context(), user.ID, chi.URLParam(r, "tagID")); err != nil {
		writeTagError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreTag(w http.ResponseWriter, r *http.Request, user *store.User) {
	tag, err := s.store.RestoreTag(r.Context(), user.ID, chi.URLParam(r, "tagID"))
	if err != nil {
		writeTagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func decodeTagInput(w http.ResponseWriter, r *http.Request) (store.TagInput, bool) {
	var input store.TagInput
	if !decodeJSONBody(w, r, &input) {
		return store.TagInput{}, false
	}
	return input, true
}

func writeTagError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidTagInput), store.IsValidation(err, store.ErrDuplicateTagName):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrTagNotFound):
		writeError(w, http.StatusNotFound, "tag_not_found", "tag not found")
	default:
		writeError(w, http.StatusInternalServerError, "tag_operation_failed", "tag operation failed")
	}
}
