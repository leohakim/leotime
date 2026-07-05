package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	cfg   config.Config
	store *store.Store
}

type sessionResponse struct {
	Authenticated bool        `json:"authenticated"`
	User          *store.User `json:"user"`
}

func NewRouter(cfg config.Config, st *store.Store) http.Handler {
	server := &Server{cfg: cfg, store: st}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/api/health", server.health)
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/session", server.session)
		r.Post("/auth/login", server.login)
		r.Post("/auth/logout", server.logout)
		r.Get("/overview", server.requireUser(server.overview))
		r.Get("/clients", server.requireUser(server.listClients))
		r.Post("/clients", server.requireUser(server.createClient))
		r.Get("/clients/{clientID}", server.requireUser(server.getClient))
		r.Patch("/clients/{clientID}", server.requireUser(server.updateClient))
		r.Delete("/clients/{clientID}", server.requireUser(server.archiveClient))
		r.Get("/projects", server.requireUser(server.listProjects))
		r.Post("/projects", server.requireUser(server.createProject))
		r.Get("/projects/{projectID}", server.requireUser(server.getProject))
		r.Patch("/projects/{projectID}", server.requireUser(server.updateProject))
		r.Delete("/projects/{projectID}", server.requireUser(server.archiveProject))
		r.Get("/tasks", server.requireUser(server.listTasks))
		r.Post("/tasks", server.requireUser(server.createTask))
		r.Get("/tasks/{taskID}", server.requireUser(server.getTask))
		r.Patch("/tasks/{taskID}", server.requireUser(server.updateTask))
		r.Delete("/tasks/{taskID}", server.requireUser(server.archiveTask))
		r.Get("/tags", server.requireUser(server.listTags))
		r.Post("/tags", server.requireUser(server.createTag))
		r.Get("/tags/{tagID}", server.requireUser(server.getTag))
		r.Patch("/tags/{tagID}", server.requireUser(server.updateTag))
		r.Delete("/tags/{tagID}", server.requireUser(server.deleteTag))
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
		r.Delete("/invoices/{invoiceID}", server.requireUser(server.deleteInvoice))
		r.Get("/invoices/{invoiceID}/export", server.requireUser(server.exportInvoice))
	})

	r.NotFound(server.notFound)
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"app":    "leotime",
		"status": "ok",
	})
}

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(r)
	if !ok {
		writeJSON(w, http.StatusOK, sessionResponse{Authenticated: false, User: nil})
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{Authenticated: true, User: user})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	user, err := s.store.Authenticate(r.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	token, expiresAt, err := s.store.CreateSession(r.Context(), user.ID, s.cfg.SessionTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create session failed")
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
		writeError(w, http.StatusInternalServerError, "load overview failed")
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) requireUser(next func(http.ResponseWriter, *http.Request, *store.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.currentUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next(w, r, user)
	}
}

func (s *Server) currentUser(r *http.Request) (*store.User, bool) {
	cookie, err := r.Cookie(s.cfg.SessionCookieName)
	if err != nil {
		return nil, false
	}
	user, err := s.store.UserBySessionToken(r.Context(), cookie.Value)
	if err != nil {
		return nil, false
	}
	return user, true
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if s.cfg.StaticDir == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if cleanPath == "." {
		cleanPath = "index.html"
	}

	fullPath := filepath.Join(s.cfg.StaticDir, cleanPath)
	info, err := os.Stat(fullPath)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, fullPath)
		return
	}

	indexPath := filepath.Join(s.cfg.StaticDir, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}
