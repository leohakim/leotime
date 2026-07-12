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
		case r.Method == http.MethodGet && r.URL.Path == "/v1/agents/bc-test/usage":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"runs": []map[string]any{{
					"id": "run-test",
					"usage": map[string]int{
						"inputTokens": 1200, "outputTokens": 340, "cacheReadTokens": 0,
						"cacheWriteTokens": 0, "totalTokens": 1540,
					},
				}},
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

	result, err := client.PromptOnce(context.Background(), "cursor-test-key", "write standup")
	if err != nil {
		t.Fatalf("PromptOnce failed: %v", err)
	}
	if !strings.Contains(result.Text, "Hasta mañana team!") {
		t.Fatalf("unexpected result: %s", result.Text)
	}
	if result.Usage.TotalTokens != 1540 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
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
		if strings.Contains(r.URL.Path, "/usage") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"runs": []map[string]any{{
					"id":    "run-test",
					"usage": map[string]int{"inputTokens": 100, "outputTokens": 50, "totalTokens": 150},
				}},
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

	result, ok := tryCursorAI(ContextBundle{
		Date:         "2026-07-12",
		TemplateText: "borrador",
		Locale:       "es",
	}, "cursor-test-key", true)
	if !ok {
		t.Fatal("expected cursor enrichment")
	}
	if !strings.Contains(result.Text, "Texto IA") {
		t.Fatalf("unexpected text: %s", result.Text)
	}
}
