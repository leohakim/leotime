package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/leotime/leotime/apps/api/internal/backup"
	"github.com/leotime/leotime/apps/api/internal/billing"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/maintenance"
	"github.com/leotime/leotime/apps/api/internal/notify"
	"github.com/leotime/leotime/apps/api/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	cfg                   config.Config
	store                 *store.Store
	passwordReset         *notify.PasswordResetService
	backups               *backup.Service
	issueService          *billing.IssueService
	documents             *billing.DocumentStore
	renderer              billing.Renderer
	loginLimiter          *fixedWindowLimiter
	forgotPasswordLimiter *fixedWindowLimiter
}

type sessionResponse struct {
	Authenticated bool        `json:"authenticated"`
	User          *store.User `json:"user"`
}

func NewRouter(cfg config.Config, st *store.Store, passwordReset *notify.PasswordResetService, backups *backup.Service) (http.Handler, error) {
	documentStore, err := billing.NewDocumentStore(cfg.DocumentRoot)
	if err != nil {
		return nil, fmt.Errorf("billing document store: %w", err)
	}
	renderer := billing.NewRenderer()
	issueService := billing.NewIssueService(st, renderer, documentStore)

	server := &Server{
		cfg:                   cfg,
		store:                 st,
		passwordReset:         passwordReset,
		backups:               backups,
		issueService:          issueService,
		documents:             documentStore,
		renderer:              renderer,
		loginLimiter:          newFixedWindowLimiter(10, 15*time.Minute),
		forgotPasswordLimiter: newFixedWindowLimiter(5, time.Hour),
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	if cfg.TrustForwardedHeaders {
		r.Use(middleware.RealIP)
	}
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(server.maintenanceMiddleware)

	r.Get("/api/health", server.health)
	r.Get("/metrics", server.metrics)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/session", server.session)
		r.Post("/auth/login", server.login)
		r.Post("/auth/logout", server.logout)
		r.Post("/auth/forgot-password", server.forgotPassword)
		r.Post("/auth/reset-password", server.resetPassword)
		r.Get("/overview", server.requireUser(server.overview))
		r.Get("/dashboard/stats", server.requireUser(server.getDashboardStats))
		r.Get("/profile", server.requireUser(server.getProfile))
		r.Patch("/profile", server.requireUser(server.updateProfile))
		r.Post("/profile/change-password", server.requireUser(server.changePassword))
		r.Get("/backups/settings", server.requireUser(server.getBackupSettings))
		r.Put("/backups/settings", server.requireUser(server.putBackupSettings))
		r.Post("/backups/test", server.requireUser(server.testBackupConnection))
		r.Post("/backups/run", server.requireUser(server.runBackup))
		r.Get("/backups/objects", server.requireUser(server.listBackupObjects))
		r.Post("/backups/restore", server.requireUser(server.restoreBackup))
		r.Get("/backups/status", server.requireUser(server.getBackupStatus))
		r.Get("/clients", server.requireUser(server.listClients))
		r.Post("/clients", server.requireUser(server.createClient))
		r.Get("/clients/{clientID}", server.requireUser(server.getClient))
		r.Patch("/clients/{clientID}", server.requireUser(server.updateClient))
		r.Delete("/clients/{clientID}", server.requireUser(server.archiveClient))
		r.Post("/clients/{clientID}/restore", server.requireUser(server.restoreClient))
		r.Get("/projects", server.requireUser(server.listProjects))
		r.Post("/projects", server.requireUser(server.createProject))
		r.Get("/projects/{projectID}", server.requireUser(server.getProject))
		r.Patch("/projects/{projectID}", server.requireUser(server.updateProject))
		r.Delete("/projects/{projectID}", server.requireUser(server.archiveProject))
		r.Post("/projects/{projectID}/restore", server.requireUser(server.restoreProject))
		r.Get("/tasks", server.requireUser(server.listTasks))
		r.Post("/tasks", server.requireUser(server.createTask))
		r.Get("/tasks/{taskID}", server.requireUser(server.getTask))
		r.Patch("/tasks/{taskID}", server.requireUser(server.updateTask))
		r.Delete("/tasks/{taskID}", server.requireUser(server.archiveTask))
		r.Post("/tasks/{taskID}/restore", server.requireUser(server.restoreTask))
		r.Get("/tags", server.requireUser(server.listTags))
		r.Post("/tags", server.requireUser(server.createTag))
		r.Get("/tags/{tagID}", server.requireUser(server.getTag))
		r.Patch("/tags/{tagID}", server.requireUser(server.updateTag))
		r.Delete("/tags/{tagID}", server.requireUser(server.archiveTag))
		r.Post("/tags/{tagID}/restore", server.requireUser(server.restoreTag))
		r.Get("/time-entries", server.requireUser(server.listTimeEntries))
		r.Post("/time-entries", server.requireUser(server.createTimeEntry))
		r.Get("/time-entries/{timeEntryID}", server.requireUser(server.getTimeEntry))
		r.Patch("/time-entries/{timeEntryID}", server.requireUser(server.updateTimeEntry))
		r.Delete("/time-entries/{timeEntryID}", server.requireUser(server.deleteTimeEntry))
		r.Get("/timers", server.requireUser(server.listTimers))
		r.Post("/timers", server.requireUser(server.startTimer))
		r.Patch("/timers/{timeEntryID}", server.requireUser(server.updateTimer))
		r.Post("/timers/{timeEntryID}/stop", server.requireUser(server.stopTimer))
		r.Delete("/timers/{timeEntryID}", server.requireUser(server.discardTimer))
		r.Get("/reports/time", server.requireUser(server.getTimeReport))
		r.Get("/reports/time/export", server.requireUser(server.exportTimeReport))
		r.Get("/invoices", server.requireUser(server.listInvoices))
		r.Post("/invoices/draft-from-time", server.requireUser(server.createInvoiceDraftFromTime))
		r.Get("/invoices/{invoiceID}", server.requireUser(server.getInvoice))
		r.Patch("/invoices/{invoiceID}", server.requireUser(server.updateInvoice))
		r.Post("/invoices/{invoiceID}/status", server.requireUser(server.updateInvoiceStatus))
		r.Post("/invoices/{invoiceID}/preview", server.requireUser(server.previewInvoice))
		r.Post("/invoices/{invoiceID}/issue", server.requireUser(server.issueInvoice))
		r.Post("/invoices/{invoiceID}/cancel", server.requireUser(server.cancelInvoice))
		r.Get("/invoices/{invoiceID}/documents", server.requireUser(server.listInvoiceDocuments))
		r.Get("/invoices/{invoiceID}/documents/{documentID}/download", server.requireUser(server.downloadInvoiceDocument))
		r.Delete("/invoices/{invoiceID}", server.requireUser(server.deleteInvoice))
		r.Get("/invoices/{invoiceID}/export", server.requireUser(server.exportInvoice))

		r.Get("/invoice-series", server.requireUser(server.listInvoiceSeries))
		r.Post("/invoice-series", server.requireUser(server.createInvoiceSeries))
		r.Patch("/invoice-series/{seriesID}", server.requireUser(server.updateInvoiceSeries))
		r.Post("/imports/solidtime", server.requireUser(server.importSolidtime))
	})

	r.NotFound(server.notFound)
	return r, nil
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"app":    "leotime",
		"status": "ok",
	})
}

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	result := s.lookupSessionUser(r)
	if result.serviceUnavailable {
		writeError(w, http.StatusServiceUnavailable, "session_lookup_failed", "session lookup failed")
		return
	}
	if result.unauthenticated {
		writeJSON(w, http.StatusOK, sessionResponse{Authenticated: false, User: nil})
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{Authenticated: true, User: result.user})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimitAuth(w, r, "login:"+s.clientIP(r)) {
		return
	}

	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSONBody(w, r, &request) {
		return
	}

	user, err := s.store.Authenticate(r.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "login_failed", "login failed")
		return
	}

	token, expiresAt, err := s.store.CreateSession(r.Context(), user.ID, s.cfg.SessionTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_create_failed", "create session failed")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure,
	})

	writeJSON(w, http.StatusOK, sessionResponse{Authenticated: true, User: user})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(s.cfg.SessionCookieName); err == nil {
		_ = s.store.DeleteSession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure,
	})

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request, user *store.User) {
	overview, err := s.store.Overview(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "overview_load_failed", "load overview failed")
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) requireUser(next func(http.ResponseWriter, *http.Request, *store.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := s.lookupSessionUser(r)
		if result.serviceUnavailable {
			writeError(w, http.StatusServiceUnavailable, "session_lookup_failed", "session lookup failed")
			return
		}
		if result.unauthenticated {
			writeError(w, http.StatusUnauthorized, "authentication_required", "authentication required")
			return
		}
		next(w, r, result.user)
	}
}

type sessionLookupResult struct {
	user               *store.User
	unauthenticated    bool
	serviceUnavailable bool
}

func (s *Server) lookupSessionUser(r *http.Request) sessionLookupResult {
	cookie, err := r.Cookie(s.cfg.SessionCookieName)
	if err != nil {
		return sessionLookupResult{unauthenticated: true}
	}

	user, err := s.store.UserBySessionToken(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			return sessionLookupResult{unauthenticated: true}
		}
		return sessionLookupResult{serviceUnavailable: true}
	}
	return sessionLookupResult{user: user}
}

func (s *Server) currentUser(r *http.Request) (*store.User, bool) {
	result := s.lookupSessionUser(r)
	if result.user == nil {
		return nil, false
	}
	return result.user, true
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	if s.cfg.StaticDir == "" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	fullPath, ok := safeStaticFilePath(s.cfg.StaticDir, r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	info, err := os.Stat(fullPath)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, fullPath)
		return
	}

	indexPath, ok := safeStaticFilePath(s.cfg.StaticDir, "index.html")
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}

	writeError(w, http.StatusNotFound, "not_found", "not found")
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	if !metricsAuthorized(s.cfg, r) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	promhttp.Handler().ServeHTTP(w, r)
}

func (s *Server) maintenanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !maintenance.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusServiceUnavailable, "maintenance_mode", "server is in maintenance mode; reload the application")
			return
		}
		next.ServeHTTP(w, r)
	})
}
