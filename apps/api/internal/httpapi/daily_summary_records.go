package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/leotime/leotime/apps/api/internal/enrich"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type dailySummaryRecordRequest struct {
	DraftText  string                    `json:"draftText"`
	ManualNote string                    `json:"manualNote"`
	Options    store.DailySummaryOptions `json:"options"`
}

type dailySummaryApproveRequest struct {
	ApprovedText string `json:"approvedText"`
}

type dailySummaryEnrichContextResponse struct {
	Date         string                    `json:"date"`
	TemplateText string                    `json:"templateText"`
	ManualNote   string                    `json:"manualNote"`
	Locale       string                    `json:"locale"`
	AuthorEmail  string                    `json:"authorEmail"`
	Projects     []enrich.ProjectWorkspace `json:"projects"`
	Record       *store.DailySummaryRecord `json:"record,omitempty"`
}

func (s *Server) getDailySummaryRecord(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	record, err := s.store.DailySummaryByDate(r.Context(), user.ID, date)
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
		Options:          body.Options,
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
		Text             string `json:"text"`
		ManualNote       string `json:"manualNote"`
		GenerationSource string `json:"generationSource"`
		ContextJSON      string `json:"contextJson"`
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

	record, err := s.store.UpsertDailySummaryDraft(r.Context(), user.ID, date, store.DailySummaryRecordInput{
		DraftText:        body.Text,
		ManualNote:       body.ManualNote,
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
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) approveDailySummaryRecord(w http.ResponseWriter, r *http.Request, user *store.User) {
	date := strings.TrimSpace(chi.URLParam(r, "date"))
	var body dailySummaryApproveRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}

	record, err := s.store.ApproveDailySummary(r.Context(), user.ID, date, body.ApprovedText)
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
	record, err := s.store.ReopenDailySummary(r.Context(), user.ID, date)
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
	if existing, err := s.store.DailySummaryByDate(r.Context(), user.ID, date); err == nil {
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

	writeJSON(w, http.StatusOK, dailySummaryEnrichContextResponse{
		Date:         date,
		TemplateText: summary.Text,
		ManualNote:   manualNote,
		Locale:       profile.Locale,
		AuthorEmail:  authorEmail,
		Projects:     workspaces,
		Record:       record,
	})
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
	return store.DailySummaryOptions{
		Date:           date,
		Timezone:       profile.Settings.Timezone,
		Locale:         profile.Locale,
		IncludeClient:  includeClient,
		IncludeProject: includeProject,
		IncludeClosing: includeClosing,
		BillableOnly:   billableOnly,
	}, true
}
