package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type tasksResponse struct {
	Tasks []store.Task `json:"tasks"`
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request, user *store.User) {
	includeArchived := r.URL.Query().Get("includeArchived") == "true"
	projectID := r.URL.Query().Get("projectId")
	tasks, err := s.store.ListTasks(r.Context(), user.ID, includeArchived, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load tasks failed")
		return
	}
	writeJSON(w, http.StatusOK, tasksResponse{Tasks: tasks})
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeTaskInput(w, r)
	if !ok {
		return
	}

	task, err := s.store.CreateTask(r.Context(), user.ID, input)
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request, user *store.User) {
	task, err := s.store.TaskByID(r.Context(), user.ID, chi.URLParam(r, "taskID"))
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeTaskInput(w, r)
	if !ok {
		return
	}

	task, err := s.store.UpdateTask(r.Context(), user.ID, chi.URLParam(r, "taskID"), input)
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) archiveTask(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.ArchiveTask(r.Context(), user.ID, chi.URLParam(r, "taskID")); err != nil {
		writeTaskError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreTask(w http.ResponseWriter, r *http.Request, user *store.User) {
	task, err := s.store.RestoreTask(r.Context(), user.ID, chi.URLParam(r, "taskID"))
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func decodeTaskInput(w http.ResponseWriter, r *http.Request) (store.TaskInput, bool) {
	var input store.TaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return store.TaskInput{}, false
	}
	return input, true
}

func writeTaskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrInvalidTaskInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, store.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "task not found")
	default:
		writeError(w, http.StatusInternalServerError, "task operation failed")
	}
}
