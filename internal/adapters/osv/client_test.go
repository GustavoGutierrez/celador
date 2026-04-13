package osv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestClientQuery_EmptyDepsSkipped(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.URL, server.URL, time.Hour)
	results, err := client.Query(context.Background(), []shared.Dependency{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d items", len(results))
	}
	if callCount != 0 {
		t.Errorf("expected no HTTP calls for empty deps, got %d", callCount)
	}
}

func TestClientQuery_NoVulnerabilities(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"vulns":[]}]}`))
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.URL, server.URL, time.Hour)
	deps := []shared.Dependency{{Name: "safe-pkg", Version: "1.0.0", Ecosystem: "npm"}}
	results, err := client.Query(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no findings for safe package, got %d", len(results))
	}
}

func TestClientQuery_WithVulnerabilities(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"vulns":[{
			"id":"GHSA-xxxx",
			"summary":"Test vulnerability",
			"severity":[{"type":"CVSS_V3","score":"7.5"}],
			"affected":[{"package":{"name":"vuln-pkg"},"ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"2.0.0"}]}]}]
		}]}]}`))
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.URL, server.URL, time.Hour)
	deps := []shared.Dependency{{Name: "vuln-pkg", Version: "1.0.0", Ecosystem: "npm"}}
	results, err := client.Query(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(results))
	}
	if results[0].ID != "GHSA-xxxx" {
		t.Errorf("expected ID 'GHSA-xxxx', got %q", results[0].ID)
	}
	if results[0].Severity != shared.SeverityHigh {
		t.Errorf("expected severity high, got %v", results[0].Severity)
	}
	if results[0].FixVersion != "2.0.0" {
		t.Errorf("expected fix version 2.0.0, got %q", results[0].FixVersion)
	}
}

func TestClientQuery_NetworkError(t *testing.T) {
	t.Parallel()
	client := NewClientWithEndpoint("http://127.0.0.1:1", "http://127.0.0.1:1", time.Hour)
	deps := []shared.Dependency{{Name: "pkg", Version: "1.0.0", Ecosystem: "npm"}}
	_, err := client.Query(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}

func TestClientQuery_HTTPError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.URL, server.URL, time.Hour)
	deps := []shared.Dependency{{Name: "pkg", Version: "1.0.0", Ecosystem: "npm"}}
	_, err := client.Query(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestParseOSVSeverity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		entries  []osvSeverityEntry
		expected shared.Severity
	}{
		{"critical_9.8", []osvSeverityEntry{{Type: "CVSS_V3", Score: "9.8"}}, shared.SeverityCritical},
		{"high_7.5", []osvSeverityEntry{{Type: "CVSS_V3", Score: "7.5"}}, shared.SeverityHigh},
		{"medium_5.3", []osvSeverityEntry{{Type: "CVSS_V3", Score: "5.3"}}, shared.SeverityMedium},
		{"low_2.1", []osvSeverityEntry{{Type: "CVSS_V3", Score: "2.1"}}, shared.SeverityLow},
		{"empty_returns_medium", []osvSeverityEntry{}, shared.SeverityMedium},
		{"non_cvss_ignored", []osvSeverityEntry{{Type: "OTHER", Score: "10.0"}}, shared.SeverityMedium},
		{"invalid_score_ignored", []osvSeverityEntry{{Type: "CVSS_V3", Score: "invalid"}}, shared.SeverityMedium},
		{"multiple_uses_first_valid", []osvSeverityEntry{
			{Type: "CVSS_V3", Score: "4.3"},
			{Type: "CVSS_V3", Score: "8.1"},
		}, shared.SeverityMedium},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOSVSeverity(tt.entries)
			if result != tt.expected {
				t.Errorf("parseOSVSeverity(%v) = %v, want %v", tt.entries, result, tt.expected)
			}
		})
	}
}
