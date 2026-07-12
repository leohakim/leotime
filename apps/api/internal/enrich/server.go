package enrich

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

type EnrichRequest struct {
	Date         string             `json:"date"`
	TemplateText string             `json:"templateText"`
	ManualNote   string             `json:"manualNote"`
	Feedback     string             `json:"feedback"`
	CurrentDraft string             `json:"currentDraft"`
	Locale       string             `json:"locale"`
	AuthorEmail  string             `json:"authorEmail"`
	Projects     []ProjectWorkspace `json:"projects"`
	CursorAPIKey string             `json:"cursorApiKey,omitempty"`
	AIEnabled    bool               `json:"aiEnabled"`
}

type EnrichResponse struct {
	Text    string        `json:"text"`
	Context ContextBundle `json:"context"`
	Source  string        `json:"source"`
}

func NewServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeCORS(w, r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /enrich", handleEnrich)
	mux.HandleFunc("OPTIONS /enrich", func(w http.ResponseWriter, r *http.Request) {
		writeCORS(w, r)
		w.WriteHeader(http.StatusNoContent)
	})
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			writeCORS(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleEnrich(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, r)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	var request EnrichRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Date) == "" {
		http.Error(w, "date is required", http.StatusBadRequest)
		return
	}

	commits, _ := CollectGitCommits(request.Date, request.AuthorEmail, request.Projects)
	cursorActivity, _ := CollectCursorActivity(request.Date, request.Projects)
	bundle := ContextBundle{
		Date:           request.Date,
		TemplateText:   request.TemplateText,
		ManualNote:     request.ManualNote,
		Feedback:       request.Feedback,
		CurrentDraft:   request.CurrentDraft,
		Commits:        commits,
		CursorActivity: cursorActivity,
		Locale:         request.Locale,
	}

	text := BuildEnrichedText(bundle)
	source := "context"
	if aiText, ok := tryCursorAI(bundle, request.CursorAPIKey, request.AIEnabled); ok {
		text = aiText
		source = "cursor"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(EnrichResponse{
		Text:    text,
		Context: bundle,
		Source:  source,
	}); err != nil {
		log.Printf("encode enrich response: %v", err)
	}
}

func writeCORS(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if allowedOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func allowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, candidate := range []string{
		"http://127.0.0.1:5173",
		"http://localhost:5173",
		"http://127.0.0.1:8080",
		"http://localhost:8080",
		"http://127.0.0.1:3000",
		"http://localhost:3000",
	} {
		if origin == candidate {
			return true
		}
	}
	return strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://localhost:")
}
