package enrich

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildCursorPromptIncludesFacts(t *testing.T) {
	prompt := BuildCursorPrompt(ContextBundle{
		Date:         "2026-07-12",
		Locale:       "es",
		TemplateText: "12/7:\nResumen de hoy:\nPor la mañana avancé con RTVE.",
		ManualNote:   "Quedó pendiente el deploy.",
		Commits: []CommitLine{{
			ProjectName: "leotime",
			Hash:        "abc1234",
			Subject:     "add daily summary workflow",
		}},
	})
	if !containsAll(prompt, "Resumen de entradas de tiempo", "abc1234", "Quedó pendiente el deploy.", "Hasta mañana team!") {
		t.Fatalf("unexpected prompt: %s", prompt)
	}
}

func TestCursorClientPromptOnce(t *testing.T) {
	var createBody cursorCreateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/agents":
			user, pass, ok := r.BasicAuth()
			if !ok || user != "cursor-test-key" || pass != "" {
				t.Fatalf("unexpected auth: %s", r.Header.Get("Authorization"))
			}
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agent": map[string]string{"id": "bc-test"},
				"run":   map[string]string{"id": "run-test"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/agents/bc-test/runs/run-test":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "FINISHED",
				"result": "12/7:\nResumen de hoy:\nHoy cerré el enricher.\nHasta mañana team!",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := &CursorClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		PollEvery:  10 * time.Millisecond,
		Timeout:    time.Second,
	}

	text, err := client.PromptOnce(context.Background(), "cursor-test-key", "write standup")
	if err != nil {
		t.Fatalf("PromptOnce failed: %v", err)
	}
	if !strings.Contains(text, "Hasta mañana team!") {
		t.Fatalf("unexpected result: %s", text)
	}
	if !strings.Contains(createBody.Prompt.Text, "write standup") {
		t.Fatalf("unexpected prompt body: %+v", createBody)
	}
}

func TestTryCursorAIUsesInjectedKey(t *testing.T) {
	original := cursorPromptClient
	defer func() { cursorPromptClient = original }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/agents" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agent": map[string]string{"id": "bc-test"},
				"run":   map[string]string{"id": "run-test"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "FINISHED",
			"result": "12/7:\nResumen de hoy:\nTexto IA.\nHasta mañana team!",
		})
	}))
	defer server.Close()

	cursorPromptClient = &CursorClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		PollEvery:  10 * time.Millisecond,
		Timeout:    time.Second,
	}

	text, ok := tryCursorAI(ContextBundle{
		Date:         "2026-07-12",
		TemplateText: "borrador",
		Locale:       "es",
	}, "cursor-test-key", true)
	if !ok {
		t.Fatal("expected cursor enrichment")
	}
	if !strings.Contains(text, "Texto IA") {
		t.Fatalf("unexpected text: %s", text)
	}
}
