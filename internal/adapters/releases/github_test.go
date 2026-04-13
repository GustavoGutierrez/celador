package releases

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubLatest_ReleaseFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("expected Accept header 'application/vnd.github+json', got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v0.3.1","name":"v0.3.1","published_at":"2026-04-13T00:00:00Z"}`))
	}))
	defer server.Close()

	source := NewGitHubLatestReleaseSourceWithEndpoint(server.URL, time.Second)
	tag, err := source.Latest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v0.3.1" {
		t.Errorf("expected tag 'v0.3.1', got %q", tag)
	}
}

func TestGitHubLatest_NoReleases(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer server.Close()

	source := NewGitHubLatestReleaseSourceWithEndpoint(server.URL, time.Second)
	_, err := source.Latest(context.Background())
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

func TestGitHubLatest_NetworkError(t *testing.T) {
	t.Parallel()
	source := NewGitHubLatestReleaseSourceWithEndpoint("http://127.0.0.1:1", time.Second)
	_, err := source.Latest(context.Background())
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}

func TestGitHubLatest_InvalidResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	source := NewGitHubLatestReleaseSourceWithEndpoint(server.URL, time.Second)
	_, err := source.Latest(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
}

func TestGitHubLatest_HTTPError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	source := NewGitHubLatestReleaseSourceWithEndpoint(server.URL, time.Second)
	_, err := source.Latest(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

// NewGitHubLatestReleaseSourceWithEndpoint allows overriding the endpoint for testing.
func NewGitHubLatestReleaseSourceWithEndpoint(endpoint string, timeout time.Duration) *GitHubLatestReleaseSource {
	return &GitHubLatestReleaseSource{
		client:   &http.Client{Timeout: timeout},
		endpoint: endpoint,
	}
}
