package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/store"
	"github.com/leotime/leotime/apps/api/internal/vcs"
)

type vcsConnectionRequest struct {
	Provider        string
	BaseURL         string
	OwnerIdentity   string
	DefaultClientID string
	Token           string
}

type vcsRepositoryRequest struct {
	ConnectionID string
	Owner        string
	Name         string
	ClientID     string
	ProjectID    string
}

func (s *Server) listVCSConnections(w http.ResponseWriter, r *http.Request, user *store.User) {
	connections, err := s.store.ListVCSConnections(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vcs_connections_load_failed", "load vcs connections failed")
		return
	}
	items := make([]map[string]any, 0, len(connections))
	for _, connection := range connections {
		items = append(items, vcsConnectionPayload(connection))
	}
	writeJSON(w, http.StatusOK, map[string]any{"connections": items})
}

func (s *Server) createVCSConnection(w http.ResponseWriter, r *http.Request, user *store.User) {
	var body vcsConnectionRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if err := s.validateVCSConnectionHost(body.BaseURL); err != nil {
		writeError(w, http.StatusBadRequest, "vcs_host_not_allowed", "VCS host is not allowed")
		return
	}
	tokenEnc := ""
	if strings.TrimSpace(body.Token) != "" {
		encoded, err := s.encryptSecret(strings.TrimSpace(body.Token))
		if err != nil {
			writeAISettingsError(w, err)
			return
		}
		tokenEnc = encoded
	}
	connection, err := s.store.CreateVCSConnection(r.Context(), user.ID, store.VCSConnectionInput{
		Provider: body.Provider, BaseURL: body.BaseURL, OwnerIdentity: body.OwnerIdentity, DefaultClientID: body.DefaultClientID,
	}, tokenEnc)
	if err != nil {
		writeVCSError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, vcsConnectionPayload(*connection))
}

func (s *Server) updateVCSConnection(w http.ResponseWriter, r *http.Request, user *store.User) {
	var body vcsConnectionRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if err := s.validateVCSConnectionHost(body.BaseURL); err != nil {
		writeError(w, http.StatusBadRequest, "vcs_host_not_allowed", "VCS host is not allowed")
		return
	}
	tokenEnc := ""
	if strings.TrimSpace(body.Token) != "" {
		encoded, err := s.encryptSecret(strings.TrimSpace(body.Token))
		if err != nil {
			writeAISettingsError(w, err)
			return
		}
		tokenEnc = encoded
	}
	connection, err := s.store.UpdateVCSConnection(r.Context(), user.ID, chi.URLParam(r, "connectionID"), store.VCSConnectionInput{
		Provider: body.Provider, BaseURL: body.BaseURL, OwnerIdentity: body.OwnerIdentity, DefaultClientID: body.DefaultClientID,
	}, tokenEnc)
	if err != nil {
		writeVCSError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vcsConnectionPayload(*connection))
}

func (s *Server) deleteVCSConnection(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.DeleteVCSConnection(r.Context(), user.ID, chi.URLParam(r, "connectionID")); err != nil {
		writeVCSError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testVCSConnection(w http.ResponseWriter, r *http.Request, user *store.User) {
	baseURL, _, tokenEnc, err := s.store.VCSConnectionCredential(r.Context(), user.ID, chi.URLParam(r, "connectionID"))
	if err != nil {
		writeVCSError(w, err)
		return
	}
	token, err := s.decryptSecret(tokenEnc)
	if err != nil {
		writeAISettingsError(w, err)
		return
	}
	if strings.TrimSpace(token) == "" {
		writeError(w, http.StatusBadRequest, "vcs_token_missing", "VCS token is not configured")
		return
	}
	if err := newVCSProvider(baseURL).TestConnection(r.Context(), token); err != nil {
		writeVCSProviderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "connection ok"})
}

func (s *Server) previewVCSConnectionContext(w http.ResponseWriter, r *http.Request, user *store.User) {
	connectionID := chi.URLParam(r, "connectionID")
	baseURL, ownerIdentity, tokenEnc, err := s.store.VCSConnectionCredential(r.Context(), user.ID, connectionID)
	if err != nil {
		writeVCSError(w, err)
		return
	}
	repositories, err := s.store.ListVCSRepositories(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vcs_repositories_load_failed", "load vcs repositories failed")
		return
	}
	linked := make([]store.VCSRepository, 0)
	for _, repository := range repositories {
		if repository.ConnectionID == connectionID && repository.Enabled {
			linked = append(linked, repository)
		}
	}
	s.writeVCSContextPreview(w, r, baseURL, ownerIdentity, tokenEnc, linked)
}

func (s *Server) listVCSRepositories(w http.ResponseWriter, r *http.Request, user *store.User) {
	repositories, err := s.store.ListVCSRepositories(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vcs_repositories_load_failed", "load vcs repositories failed")
		return
	}
	items := make([]map[string]any, 0, len(repositories))
	for _, repository := range repositories {
		items = append(items, vcsRepositoryPayload(repository))
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositories": items})
}

func (s *Server) createVCSRepository(w http.ResponseWriter, r *http.Request, user *store.User) {
	var body vcsRepositoryRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	repository, err := s.store.UpsertVCSRepository(r.Context(), user.ID, store.VCSRepositoryInput{
		ConnectionID: body.ConnectionID, Owner: body.Owner, Name: body.Name, ClientID: body.ClientID, ProjectID: body.ProjectID,
	})
	if err != nil {
		writeVCSError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, vcsRepositoryPayload(*repository))
}

func (s *Server) updateVCSRepository(w http.ResponseWriter, r *http.Request, user *store.User) {
	var body vcsRepositoryRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	repository, err := s.store.UpdateVCSRepository(r.Context(), user.ID, chi.URLParam(r, "repositoryID"), store.VCSRepositoryInput{
		ConnectionID: body.ConnectionID, Owner: body.Owner, Name: body.Name, ClientID: body.ClientID, ProjectID: body.ProjectID,
	})
	if err != nil {
		writeVCSError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vcsRepositoryPayload(*repository))
}

func (s *Server) deleteVCSRepository(w http.ResponseWriter, r *http.Request, user *store.User) {
	if err := s.store.DeleteVCSRepository(r.Context(), user.ID, chi.URLParam(r, "repositoryID")); err != nil {
		writeVCSError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) previewVCSRepositoryContext(w http.ResponseWriter, r *http.Request, user *store.User) {
	repository, err := s.store.VCSRepositoryByID(r.Context(), user.ID, chi.URLParam(r, "repositoryID"))
	if err != nil {
		writeVCSError(w, err)
		return
	}
	baseURL, ownerIdentity, tokenEnc, err := s.store.VCSConnectionCredential(r.Context(), user.ID, repository.ConnectionID)
	if err != nil {
		writeVCSError(w, err)
		return
	}
	s.writeVCSContextPreview(w, r, baseURL, ownerIdentity, tokenEnc, []store.VCSRepository{*repository})
}

func (s *Server) writeVCSContextPreview(w http.ResponseWriter, r *http.Request, baseURL, ownerIdentity, tokenEnc string, repositories []store.VCSRepository) {
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_date", "date must be YYYY-MM-DD")
		return
	}
	if len(repositories) == 0 {
		writeJSON(w, http.StatusOK, vcsContextPreviewPayload(date, "not_configured", 0, vcs.Context{}))
		return
	}
	token, err := s.decryptSecret(tokenEnc)
	if err != nil {
		writeAISettingsError(w, err)
		return
	}
	if strings.TrimSpace(token) == "" {
		writeError(w, http.StatusBadRequest, "vcs_token_missing", "VCS token is not configured")
		return
	}
	repos := make([]vcs.Repository, 0, len(repositories))
	for _, repository := range repositories {
		repos = append(repos, vcs.Repository{Owner: repository.Owner, Name: repository.Name})
	}
	collected, err := newVCSProvider(baseURL).CollectDayContext(r.Context(), vcs.CollectionRequest{
		Date: date, OwnerIdentity: ownerIdentity, Token: token, Repositories: repos,
	})
	if err != nil {
		writeVCSProviderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vcsContextPreviewPayload(date, "ready", len(repositories), collected))
}

func vcsContextPreviewPayload(date, status string, repositoryCount int, collected vcs.Context) map[string]any {
	return map[string]any{
		"ok":              true,
		"status":          status,
		"date":            date,
		"repositoryCount": repositoryCount,
		"counts": map[string]int{
			"commits":      len(collected.Commits),
			"pullRequests": len(collected.PullRequests),
			"reviews":      len(collected.Reviews),
			"issues":       len(collected.Issues),
		},
		"vcs": collected,
	}
}

func (s *Server) validateVCSConnectionHost(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" {
		return errors.New("invalid host")
	}
	if !s.cfg.Production {
		return nil
	}
	if parsed.Scheme != "https" {
		return errors.New("https required")
	}
	host := strings.ToLower(parsed.Host)
	for _, allowed := range s.cfg.VCSAllowedHosts {
		if host == allowed {
			return nil
		}
	}
	return errors.New("host not allowlisted")
}

func vcsConnectionPayload(connection store.VCSConnection) map[string]any {
	return map[string]any{
		"id":              connection.ID,
		"provider":        connection.Provider,
		"baseUrl":         connection.BaseURL,
		"ownerIdentity":   connection.OwnerIdentity,
		"defaultClientId": connection.DefaultClientID,
		"tokenConfigured": connection.TokenConfigured,
		"enabled":         connection.Enabled,
		"createdAt":       connection.CreatedAt,
		"updatedAt":       connection.UpdatedAt,
	}
}

func vcsRepositoryPayload(repository store.VCSRepository) map[string]any {
	return map[string]any{
		"id":                repository.ID,
		"connectionId":      repository.ConnectionID,
		"owner":             repository.Owner,
		"name":              repository.Name,
		"clientId":          repository.ClientID,
		"effectiveClientId": repository.EffectiveClientID,
		"projectId":         repository.ProjectID,
		"enabled":           repository.Enabled,
		"createdAt":         repository.CreatedAt,
		"updatedAt":         repository.UpdatedAt,
	}
}

func writeVCSError(w http.ResponseWriter, err error) {
	switch {
	case store.IsValidation(err, store.ErrInvalidVCSInput):
		writeValidationStoreError(w, err)
	case errors.Is(err, store.ErrVCSConnectionNotFound), errors.Is(err, store.ErrVCSRepositoryNotFound):
		writeError(w, http.StatusNotFound, "vcs_not_found", "VCS resource not found")
	default:
		writeError(w, http.StatusInternalServerError, "vcs_operation_failed", "VCS operation failed")
	}
}

func writeVCSProviderError(w http.ResponseWriter, err error) {
	switch {
	case vcs.IsUnauthorized(err):
		writeError(w, http.StatusUnauthorized, "vcs_unauthorized", "VCS credentials were rejected")
	case vcs.IsRateLimited(err):
		writeError(w, http.StatusTooManyRequests, "vcs_rate_limited", "VCS provider rate limited the request")
	case vcs.IsUnavailable(err):
		writeError(w, http.StatusBadGateway, "vcs_unavailable", "VCS provider is unavailable")
	default:
		writeError(w, http.StatusBadGateway, "vcs_repo_unreachable", "VCS repository was not found or returned an invalid response; check owner/name (e.g. ENACT/backend)")
	}
}
