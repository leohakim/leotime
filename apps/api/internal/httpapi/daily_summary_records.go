package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/enrich"
	"github.com/leotime/leotime/apps/api/internal/store"
	"github.com/leotime/leotime/apps/api/internal/vcs"
)

var newVCSProvider = func(baseURL string) vcs.Provider {
	return vcs.NewGiteaProvider(baseURL, nil)
}

type dailySummaryRecordRequest struct {
	DraftText  string                     `json:"draftText"`
	ManualNote string                     `json:"manualNote"`
	Options    dailySummaryOptionsPayload `json:"options"`
}

type dailySummaryApproveRequest struct {
	ApprovedText string `json:"approvedText"`
}

type dailySummaryAIUsagePayload struct {
	InputTokens      int    `json:"inputTokens"`
	OutputTokens     int    `json:"outputTokens"`
	CacheReadTokens  int    `json:"cacheReadTokens"`
	CacheWriteTokens int    `json:"cacheWriteTokens"`
	TotalTokens      int    `json:"totalTokens"`
	ModelID          string `json:"modelId"`
}

type dailySummaryEnrichContextResponse struct {
	Date         string                    `json:"date"`
	TemplateText string                    `json:"templateText"`
	ManualNote   string                    `json:"manualNote"`
	Locale       string                    `json:"locale"`
	AuthorEmail  string                    `json:"authorEmail"`
	Projects     []enrich.ProjectWorkspace `json:"projects"`
	EntryFacts   []enrich.TimeEntryFact    `json:"entryFacts"`
	VCS          vcs.Context               `json:"vcs"`
	VCSStatus    string                    `json:"vcsStatus"`
	Record       *store.DailySummaryRecord `json:"record,omitempty"`
}

func (s *Server) listDailySummaryRecords(w http.ResponseWriter, r *http.Request, user *store.User) {
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	clientID, projectID := dailySummaryScopeFromQuery(r)
	allScopes := strings.EqualFold(r.URL.Query().Get("allScopes"), "true")

	items, err := s.store.ListDailySummaryIndex(r.Context(), user.ID, from, to, clientID, projectID, allScopes)
	if err != nil {
		if store.IsValidation(err, store.ErrInvalidTimeEntryInput) {
			writeValidationStoreError(w, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_list_failed", "list daily summaries failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getDailySummaryAIUsage(w http.ResponseWriter, r *http.Request, user *store.User) {
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	summary, err := s.store.SummarizeDailySummaryAIUsage(r.Context(), user.ID, from, to)
	if err != nil {
		if store.IsValidation(err, store.ErrInvalidTimeEntryInput) {
			writeValidationStoreError(w, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_ai_usage_failed", "load daily summary ai usage failed")
		return
	}
	runs, _, err := s.store.ListDailySummaryAIRuns(r.Context(), user.ID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "daily_summary_ai_usage_failed", "load daily summary ai usage failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": summary,
		"runs":    runs,
	})
}

func (s *Server) getDailySummaryRecord(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	clientID, projectID := dailySummaryScopeFromQuery(r)
	record, err := s.store.DailySummaryByScope(r.Context(), user.ID, date, clientID, projectID)
	if err != nil {
		if errors.Is(err, store.ErrDailySummaryNotFound) {
			writeError(w, http.StatusNotFound, "daily_summary_not_found", "daily summary not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_load_failed", "load daily summary failed")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) putDailySummaryRecord(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	var body dailySummaryRecordRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}

	record, err := s.store.UpsertDailySummaryDraft(r.Context(), user.ID, date, store.DailySummaryRecordInput{
		DraftText:        body.DraftText,
		ManualNote:       body.ManualNote,
		Options:          body.Options.toStore(date),
		GenerationSource: "manual",
		IncrementCount:   false,
	})
	if err != nil {
		if errors.Is(err, store.ErrDailySummaryApproved) {
			writeError(w, http.StatusConflict, "daily_summary_approved", "daily summary is approved")
			return
		}
		if store.IsValidation(err, store.ErrInvalidTimeEntryInput) {
			writeValidationStoreError(w, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_save_failed", "save daily summary failed")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) generateDailySummaryTemplate(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	profile, err := s.store.ProfileByUserID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_load_failed", "load profile failed")
		return
	}

	options, ok := s.parseDailySummaryOptionsFromQuery(w, r, profile)
	if !ok {
		return
	}
	options.Date = date

	var body struct {
		ManualNote string `json:"manualNote"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	options.ManualNote = strings.TrimSpace(body.ManualNote)

	summary, err := s.store.BuildDailySummary(r.Context(), user.ID, options)
	if err != nil {
		if store.IsValidation(err, store.ErrInvalidProfileInput) || store.IsValidation(err, store.ErrInvalidTimeEntryInput) {
			writeValidationStoreError(w, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_failed", "build daily summary failed")
		return
	}

	record, err := s.store.UpsertDailySummaryDraft(r.Context(), user.ID, date, store.DailySummaryRecordInput{
		DraftText:        summary.Text,
		ManualNote:       body.ManualNote,
		Options:          options,
		GenerationSource: "template",
		IncrementCount:   true,
	})
	if err != nil {
		if errors.Is(err, store.ErrDailySummaryApproved) {
			writeError(w, http.StatusConflict, "daily_summary_approved", "daily summary is approved")
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_save_failed", "save daily summary failed")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) applyDailySummaryEnrichment(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	var body struct {
		Text             string                      `json:"text"`
		ManualNote       string                      `json:"manualNote"`
		GenerationSource string                      `json:"generationSource"`
		ContextJSON      string                      `json:"contextJson"`
		ModelID          string                      `json:"modelId"`
		AIUsage          *dailySummaryAIUsagePayload `json:"aiUsage"`
		Options          dailySummaryOptionsPayload  `json:"options"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Text) == "" {
		writeError(w, http.StatusBadRequest, "daily_summary_text_required", "text is required")
		return
	}

	source := strings.TrimSpace(body.GenerationSource)
	if source == "" {
		source = "context"
	}
	options := body.Options.toStore(date)
	clientID, projectID := store.NormalizeDailySummaryScope(options.ClientID, options.ProjectID)
	if clientID == "" && projectID == "" {
		clientID, projectID = dailySummaryScopeFromQuery(r)
		options.ClientID = clientID
		options.ProjectID = projectID
	}

	record, err := s.store.UpsertDailySummaryDraft(r.Context(), user.ID, date, store.DailySummaryRecordInput{
		DraftText:        body.Text,
		ManualNote:       body.ManualNote,
		Options:          options,
		GenerationSource: source,
		ContextJSON:      body.ContextJSON,
		IncrementCount:   true,
	})
	if err != nil {
		if errors.Is(err, store.ErrDailySummaryApproved) {
			writeError(w, http.StatusConflict, "daily_summary_approved", "daily summary is approved")
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_save_failed", "save daily summary failed")
		return
	}

	if source == "cursor" && body.AIUsage != nil {
		settings, _ := s.store.AISettingsByUserID(r.Context(), user.ID)
		costPerMillion := 2.0
		if settings != nil && settings.CursorCostPerMillionUSD > 0 {
			costPerMillion = settings.CursorCostPerMillionUSD
		}
		modelID := strings.TrimSpace(body.ModelID)
		if modelID == "" {
			modelID = strings.TrimSpace(body.AIUsage.ModelID)
		}
		if _, err := s.store.InsertDailySummaryAIRun(r.Context(), user.ID, store.DailySummaryAIRunInput{
			SummaryDate:      date,
			ClientID:         clientID,
			ProjectID:        projectID,
			RecordID:         record.ID,
			ModelID:          modelID,
			Source:           source,
			InputTokens:      body.AIUsage.InputTokens,
			OutputTokens:     body.AIUsage.OutputTokens,
			CacheReadTokens:  body.AIUsage.CacheReadTokens,
			CacheWriteTokens: body.AIUsage.CacheWriteTokens,
			TotalTokens:      body.AIUsage.TotalTokens,
		}, costPerMillion); err != nil {
			writeError(w, http.StatusInternalServerError, "daily_summary_ai_usage_save_failed", "save daily summary ai usage failed")
			return
		}
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) approveDailySummaryRecord(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	clientID, projectID := dailySummaryScopeFromQuery(r)
	var body dailySummaryApproveRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}

	record, err := s.store.ApproveDailySummary(r.Context(), user.ID, date, clientID, projectID, body.ApprovedText)
	if err != nil {
		if errors.Is(err, store.ErrDailySummaryNotFound) {
			writeError(w, http.StatusNotFound, "daily_summary_not_found", "daily summary not found")
			return
		}
		if store.IsValidation(err, store.ErrInvalidTimeEntryInput) {
			writeValidationStoreError(w, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_approve_failed", "approve daily summary failed")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) reopenDailySummaryRecord(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	clientID, projectID := dailySummaryScopeFromQuery(r)
	record, err := s.store.ReopenDailySummary(r.Context(), user.ID, date, clientID, projectID)
	if err != nil {
		if errors.Is(err, store.ErrDailySummaryNotFound) {
			writeError(w, http.StatusNotFound, "daily_summary_not_found", "daily summary not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_reopen_failed", "reopen daily summary failed")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) getDailySummaryEnrichContext(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	profile, err := s.store.ProfileByUserID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_load_failed", "load profile failed")
		return
	}

	options, ok := s.parseDailySummaryOptionsFromQuery(w, r, profile)
	if !ok {
		return
	}
	options.Date = date

	var record *store.DailySummaryRecord
	if existing, err := s.store.DailySummaryByScope(r.Context(), user.ID, date, options.ClientID, options.ProjectID); err == nil {
		record = existing
	}

	manualNote := strings.TrimSpace(r.URL.Query().Get("manualNote"))
	if manualNote == "" && record != nil {
		manualNote = strings.TrimSpace(record.ManualNote)
	}
	options.ManualNote = manualNote

	summary, err := s.store.BuildDailySummary(r.Context(), user.ID, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "daily_summary_failed", "build daily summary failed")
		return
	}

	projects, err := s.store.ListProjects(r.Context(), user.ID, true, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "projects_load_failed", "load projects failed")
		return
	}

	workspaces := make([]enrich.ProjectWorkspace, 0, len(projects))
	for _, project := range projects {
		if options.ProjectID != "" && project.ID != options.ProjectID {
			continue
		}
		if options.ClientID != "" && project.ClientID != options.ClientID {
			continue
		}
		if project.LocalRepoPath == "" && project.CursorWorkspaceSlug == "" {
			continue
		}
		workspaces = append(workspaces, enrich.ProjectWorkspace{
			ProjectID:           project.ID,
			ProjectName:         project.Name,
			LocalRepoPath:       project.LocalRepoPath,
			CursorWorkspaceSlug: project.CursorWorkspaceSlug,
		})
	}

	aiSettings, _ := s.store.AISettingsByUserID(r.Context(), user.ID)
	authorEmail := profile.Email
	if aiSettings != nil && strings.TrimSpace(aiSettings.GitAuthorEmail) != "" {
		authorEmail = aiSettings.GitAuthorEmail
	}
	vcsContext, vcsStatus := s.collectDailySummaryVCS(r.Context(), date, user.ID, options.ClientID, options.ProjectID, authorEmail)
	writeJSON(w, http.StatusOK, dailySummaryEnrichContextResponse{
		Date:         date,
		TemplateText: summary.Text,
		ManualNote:   manualNote,
		Locale:       profile.Locale,
		AuthorEmail:  authorEmail,
		Projects:     workspaces,
		EntryFacts:   mapDailySummaryEntryFacts(summary.EntryFacts),
		VCS:          vcsContext,
		VCSStatus:    vcsStatus,
		Record:       record,
	})
}

func (s *Server) collectDailySummaryVCS(ctx context.Context, date, userID, clientID, projectID, fallbackIdentity string) (vcs.Context, string) {
	links, err := s.store.ListVCSCollectionLinks(ctx, userID, clientID, projectID)
	if err != nil {
		return vcs.Context{}, "unavailable"
	}
	if len(links) == 0 {
		return vcs.Context{}, "not_configured"
	}

	result := vcs.Context{}
	status := "ready"
	for _, link := range links {
		if strings.TrimSpace(link.TokenEnc) == "" {
			status = "partial"
			continue
		}
		token, err := s.decryptSecret(link.TokenEnc)
		if err != nil || strings.TrimSpace(token) == "" {
			status = "partial"
			continue
		}
		identity := strings.TrimSpace(link.Connection.OwnerIdentity)
		if identity == "" {
			identity = strings.TrimSpace(fallbackIdentity)
		}
		collected, err := newVCSProvider(link.Connection.BaseURL).CollectDayContext(ctx, vcs.CollectionRequest{
			Date:          date,
			OwnerIdentity: identity,
			Token:         token,
			Repositories:  []vcs.Repository{{Owner: link.Repository.Owner, Name: link.Repository.Name}},
		})
		if err != nil {
			status = "partial"
			continue
		}
		result.Commits = append(result.Commits, collected.Commits...)
		result.PullRequests = append(result.PullRequests, collected.PullRequests...)
		result.Reviews = append(result.Reviews, collected.Reviews...)
		result.Issues = append(result.Issues, collected.Issues...)
	}
	return result, status
}

func mapDailySummaryEntryFacts(facts []store.DailySummaryEntryFact) []enrich.TimeEntryFact {
	mapped := make([]enrich.TimeEntryFact, 0, len(facts))
	for _, fact := range facts {
		mapped = append(mapped, enrich.TimeEntryFact{
			ClientName:  fact.ClientName,
			ProjectName: fact.ProjectName,
			TaskName:    fact.TaskName,
			Topics:      append([]string(nil), fact.Topics...),
			Description: fact.Description,
		})
	}
	return mapped
}

func (s *Server) parseDailySummaryOptionsFromQuery(w http.ResponseWriter, r *http.Request, profile *store.Profile) (store.DailySummaryOptions, bool) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	if date == "" {
		date = strings.TrimSpace(r.URL.Query().Get("date"))
	}
	if date == "" {
		writeError(w, http.StatusBadRequest, "date_required", "date is required")
		return store.DailySummaryOptions{}, false
	}
	includeClient := !strings.EqualFold(r.URL.Query().Get("includeClient"), "false")
	includeProject := !strings.EqualFold(r.URL.Query().Get("includeProject"), "false")
	includeClosing := !strings.EqualFold(r.URL.Query().Get("includeClosing"), "false")
	billableOnly := strings.EqualFold(r.URL.Query().Get("billableOnly"), "true")
	clientID, projectID := dailySummaryScopeFromQuery(r)
	return store.DailySummaryOptions{
		Date:           date,
		Timezone:       profile.Settings.Timezone,
		Locale:         profile.Locale,
		IncludeClient:  includeClient,
		IncludeProject: includeProject,
		IncludeClosing: includeClosing,
		BillableOnly:   billableOnly,
		ClientID:       clientID,
		ProjectID:      projectID,
	}, true
}

func dailySummaryScopeFromQuery(r *http.Request) (string, string) {
	return store.NormalizeDailySummaryScope(
		r.URL.Query().Get("clientId"),
		r.URL.Query().Get("projectId"),
	)
}
