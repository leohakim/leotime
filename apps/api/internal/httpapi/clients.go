package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type clientsResponse struct {
	Clients []store.Client `json:"clients"`
}

func (s *Server) listClients(w http.ResponseWriter, r *http.Request, user *store.User) {
	includeArchived := r.URL.Query().Get("includeArchived") == "true"
	clients, err := s.store.ListClients(r.Context(), user.ID, includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "clients_load_failed", "load clients failed")
		return
	}
	writeJSON(w, http.StatusOK, clientsResponse{Clients: clients})
}

func (s *Server) createClient(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeClientInput(w, r)
	if !ok {
		return
	}

	client, err := s.store.CreateClient(r.Context(), user.ID, input)
	if err != nil {
		writeClientError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, client)
}

func (s *Server) getClient(w http.ResponseWriter, r *http.Request, user *store.User) {
	client, err := s.store.ClientByID(r.Context(), user.ID, chi.URLParam(r, "clientID"))
	if err != nil {
		writeClientError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, client)
}

func (s *Server) updateClient(w http.ResponseWriter, r *http.Request, user *store.User) {
	input, ok := decodeClientInput(w, r)
	if !ok {
		return
	}

	client, err := s.store.UpdateClient(r.Context(), user.ID, chi.URLParam(r, "clientID"), input)
	if err != nil {
		writeClientError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, client)
}

func (s *Server) archiveClient(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.ArchiveClient(r.Context(), user.ID, chi.URLParam(r, "clientID")); err != nil {
		writeClientError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreClient(w http.ResponseWriter, r *http.Request, user *store.User) {
	client, err := s.store.RestoreClient(r.Context(), user.ID, chi.URLParam(r, "clientID"))
	if err != nil {
		writeClientError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, client)
}

func decodeClientInput(w http.ResponseWriter, r *http.Request) (store.ClientInput, bool) {
	var input store.ClientInput
	if !decodeJSONBody(w, r, &input) {
		return store.ClientInput{}, false
	}
	return input, true
}

func writeClientError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidClientInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrClientNotFound):
		writeError(w, http.StatusNotFound, "client_not_found", "client not found")
	default:
		writeError(w, http.StatusInternalServerError, "client_operation_failed", "client operation failed")
	}
}
