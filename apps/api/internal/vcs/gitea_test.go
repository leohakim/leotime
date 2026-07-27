package vcs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGiteaCollectDayContextNormalizesRepositoryFacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token read-only-token" {
			t.Fatalf("expected Gitea token header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/repos/osoigo/backend/commits":
			_, _ = w.Write([]byte("[{\"sha\":\"abcdef123456\",\"commit\":{\"message\":\"fix auth redirect\",\"author\":{\"name\":\"Leo\",\"date\":\"2026-07-27T10:00:00Z\"}}}]"))
		case "/api/v1/repos/osoigo/backend/pulls":
			_, _ = w.Write([]byte("[{\"number\":12,\"title\":\"ship auth redirect\",\"state\":\"closed\",\"merged\":true,\"updated_at\":\"2026-07-27T11:00:00Z\",\"user\":{\"login\":\"leo\"},\"labels\":[{\"name\":\"release\"}]}]"))
		case "/api/v1/repos/osoigo/backend/pulls/12/reviews":
			_, _ = w.Write([]byte("[{\"id\":4,\"state\":\"APPROVED\",\"submitted_at\":\"2026-07-27T11:10:00Z\",\"user\":{\"login\":\"reviewer\"}}]"))
		case "/api/v1/repos/osoigo/backend/issues":
			_, _ = w.Write([]byte("[{\"number\":55,\"title\":\"auth rollout\",\"state\":\"closed\",\"updated_at\":\"2026-07-27T11:00:00Z\"}]"))
		default:
			t.Fatalf("unexpected Gitea request %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := NewGiteaProvider(server.URL, server.Client())
	context, err := provider.CollectDayContext(context.Background(), CollectionRequest{
		Date:          "2026-07-27",
		OwnerIdentity: "leo",
		Token:         "read-only-token",
		Repositories:  []Repository{{Owner: "osoigo", Name: "backend"}},
	})
	if err != nil {
		t.Fatalf("collect context: %v", err)
	}
	if len(context.Commits) != 1 || context.Commits[0].Hash != "abcdef1" || context.Commits[0].Subject != "fix auth redirect" {
		t.Fatalf("unexpected commits: %+v", context.Commits)
	}
	if len(context.PullRequests) != 1 || !context.PullRequests[0].Merged || context.PullRequests[0].Labels[0] != "release" {
		t.Fatalf("unexpected pull requests: %+v", context.PullRequests)
	}
	if len(context.Reviews) != 1 || context.Reviews[0].Reviewer != "reviewer" {
		t.Fatalf("unexpected reviews: %+v", context.Reviews)
	}
	if len(context.Issues) != 1 || context.Issues[0].Title != "auth rollout" {
		t.Fatalf("unexpected issues: %+v", context.Issues)
	}
	if strings.Contains(context.PromptFacts(), "abcdef123456") {
		t.Fatalf("expected capped facts, got %q", context.PromptFacts())
	}
}

func TestGiteaCollectDayContextExcludesFactsOutsideRequestedDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/repos/osoigo/backend/commits":
			_, _ = w.Write([]byte("[{\"sha\":\"today123\",\"commit\":{\"message\":\"today\",\"author\":{\"name\":\"leo\",\"date\":\"2026-07-27T10:00:00Z\"}}},{\"sha\":\"older123\",\"commit\":{\"message\":\"older\",\"author\":{\"name\":\"leo\",\"date\":\"2026-07-26T10:00:00Z\"}}}]"))
		case "/api/v1/repos/osoigo/backend/pulls":
			_, _ = w.Write([]byte("[{\"number\":12,\"title\":\"today PR\",\"state\":\"open\",\"updated_at\":\"2026-07-27T10:00:00Z\"},{\"number\":11,\"title\":\"older PR\",\"state\":\"closed\",\"updated_at\":\"2026-07-26T10:00:00Z\"}]"))
		case "/api/v1/repos/osoigo/backend/pulls/12/reviews":
			_, _ = w.Write([]byte("[{\"id\":4,\"state\":\"APPROVED\",\"submitted_at\":\"2026-07-27T11:00:00Z\",\"user\":{\"login\":\"reviewer\"}}]"))
		case "/api/v1/repos/osoigo/backend/issues":
			_, _ = w.Write([]byte("[{\"number\":55,\"title\":\"today issue\",\"state\":\"open\",\"updated_at\":\"2026-07-27T10:00:00Z\"},{\"number\":54,\"title\":\"older issue\",\"state\":\"closed\",\"updated_at\":\"2026-07-26T10:00:00Z\"}]"))
		default:
			t.Fatalf("unexpected Gitea request %s", r.URL.Path)
		}
	}))
	defer server.Close()

	context, err := NewGiteaProvider(server.URL, server.Client()).CollectDayContext(context.Background(), CollectionRequest{
		Date: "2026-07-27", Token: "read-only-token", Repositories: []Repository{{Owner: "osoigo", Name: "backend"}},
	})
	if err != nil {
		t.Fatalf("collect context: %v", err)
	}
	if len(context.Commits) != 1 || context.Commits[0].Subject != "today" {
		t.Fatalf("unexpected filtered commits: %+v", context.Commits)
	}
	if len(context.PullRequests) != 1 || context.PullRequests[0].Number != 12 {
		t.Fatalf("unexpected filtered pull requests: %+v", context.PullRequests)
	}
	if len(context.Reviews) != 1 || context.Reviews[0].PullRequestNumber != 12 {
		t.Fatalf("unexpected filtered reviews: %+v", context.Reviews)
	}
	if len(context.Issues) != 1 || context.Issues[0].Number != 55 {
		t.Fatalf("unexpected filtered issues: %+v", context.Issues)
	}
}

func TestGiteaCollectDayContextReturnsUnauthorizedWithoutPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "credential detail must not escape", http.StatusUnauthorized)
	}))
	defer server.Close()

	provider := NewGiteaProvider(server.URL, server.Client())
	_, err := provider.CollectDayContext(context.Background(), CollectionRequest{
		Date: "2026-07-27", Token: "read-only-token", Repositories: []Repository{{Owner: "osoigo", Name: "backend"}},
	})
	if err == nil || !IsUnauthorized(err) || strings.Contains(err.Error(), "credential detail") {
		t.Fatalf("expected redacted unauthorized error, got %v", err)
	}
}

func TestGiteaTestConnectionSucceedsWithUserEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "token read-only-token" {
			t.Fatalf("expected token header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"login":"leo"}`))
	}))
	defer server.Close()

	if err := NewGiteaProvider(server.URL, server.Client()).TestConnection(context.Background(), "read-only-token"); err != nil {
		t.Fatalf("test connection: %v", err)
	}
}

func TestGiteaTestConnectionUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	err := NewGiteaProvider(server.URL, server.Client()).TestConnection(context.Background(), "bad-token")
	if err == nil || !IsUnauthorized(err) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}
