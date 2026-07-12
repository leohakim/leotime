package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxyEnricherInjectsCursorKey(t *testing.T) {
	secretsKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	router := newTestRouterWithSecretsKey(t, secretsKey)
	cookies := loginCookies(t, router)

	putBody := `{"enabled":true,"gitAuthorEmail":"dev@example.com","cursorApiKey":"cursor-proxy-key"}`
	putResponse := httptest.NewRecorder()
	putRequest := httptest.NewRequest(http.MethodPut, "/api/v1/settings/ai", strings.NewReader(putBody))
	putRequest.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		putRequest.AddCookie(cookie)
	}
	router.ServeHTTP(putResponse, putRequest)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("expected put ai settings 200, got %d: %s", putResponse.Code, putResponse.Body.String())
	}

	var received map[string]any
	enricher := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read enricher body: %v", err)
		}
		if err := json.Unmarshal(body, &received); err != nil {
			t.Fatalf("decode enricher body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"ok","source":"cursor","context":{}}`))
	}))
	defer enricher.Close()

	originalURL := localEnricherURL
	localEnricherURL = enricher.URL
	t.Cleanup(func() { localEnricherURL = originalURL })

	proxyBody := `{"date":"2026-07-12","templateText":"draft","cursorApiKey":"client-override","aiEnabled":true}`
	proxyResponse := httptest.NewRecorder()
	proxyRequest := httptest.NewRequest(http.MethodPost, "/api/v1/enricher/enrich", strings.NewReader(proxyBody))
	proxyRequest.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		proxyRequest.AddCookie(cookie)
	}
	router.ServeHTTP(proxyResponse, proxyRequest)
	if proxyResponse.Code != http.StatusOK {
		t.Fatalf("expected proxy 200, got %d: %s", proxyResponse.Code, proxyResponse.Body.String())
	}
	if received["aiEnabled"] != true {
		t.Fatalf("expected aiEnabled true, got %#v", received["aiEnabled"])
	}
	if received["cursorApiKey"] != "cursor-proxy-key" {
		t.Fatalf("expected server-injected cursor key, got %#v", received["cursorApiKey"])
	}
	if received["date"] != "2026-07-12" {
		t.Fatalf("expected date preserved, got %#v", received["date"])
	}
}
