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

func TestClientQuery_MultipleVulnsPerPackage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"vulns":[
			{"id":"GHSA-aaaa","summary":"First vuln","severity":[{"type":"CVSS_V3","score":"5.3"}],
			 "affected":[{"package":{"name":"multi-pkg"},"ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"1.5.0"}]}]}]},
			{"id":"GHSA-bbbb","summary":"Second vuln","severity":[{"type":"CVSS_V3","score":"9.8"}],
			 "affected":[{"package":{"name":"multi-pkg"},"ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"2.0.0"}]}]}]}
		]}]}`))
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.URL, server.URL, time.Hour)
	deps := []shared.Dependency{{Name: "multi-pkg", Version: "1.0.0", Ecosystem: "npm"}}
	results, err := client.Query(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(results))
	}
	if results[0].Severity != shared.SeverityMedium {
		t.Errorf("expected first severity medium, got %v", results[0].Severity)
	}
	if results[1].Severity != shared.SeverityCritical {
		t.Errorf("expected second severity critical, got %v", results[1].Severity)
	}
}

func TestClientQuery_HydratesAdvisoryFromVulnAPI(t *testing.T) {
	t.Parallel()
	batchCalled := 0
	vulnCalled := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/querybatch", func(w http.ResponseWriter, r *http.Request) {
		batchCalled++
		w.Write([]byte(`{"results":[{"vulns":[{"id":"GHSA-hydrate"}]}]}`))
	})
	mux.HandleFunc("/vulns/GHSA-hydrate", func(w http.ResponseWriter, r *http.Request) {
		vulnCalled++
		w.Write([]byte(`{
			"id": "GHSA-hydrate",
			"summary": "Hydrated advisory",
			"severity": [{"type":"CVSS_V3","score":"8.5"}],
			"affected": [{"package":{"name":"needs-hydration"},"ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"3.0.0"}]}]}]
		}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClientWithEndpoint(server.URL+"/querybatch", server.URL+"/vulns", time.Hour)
	deps := []shared.Dependency{{Name: "needs-hydration", Version: "1.0.0", Ecosystem: "npm"}}
	results, err := client.Query(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vulnCalled != 1 {
		t.Errorf("expected vuln API called once, got %d", vulnCalled)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(results))
	}
	if results[0].Summary != "Hydrated advisory" {
		t.Errorf("expected hydrated summary, got %q", results[0].Summary)
	}
	if results[0].FixVersion != "3.0.0" {
		t.Errorf("expected fix version 3.0.0, got %q", results[0].FixVersion)
	}
}

func TestClientQuery_AdvisoryHydrationFailureDoesNotBreakQuery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/querybatch", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"vulns":[{"id":"GHSA-fail-hydrate"}]}]}`))
	})
	mux.HandleFunc("/vulns/GHSA-fail-hydrate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClientWithEndpoint(server.URL+"/querybatch", server.URL+"/vulns", time.Hour)
	deps := []shared.Dependency{{Name: "pkg", Version: "1.0.0", Ecosystem: "npm"}}
	results, err := client.Query(context.Background(), deps)
	if err != nil {
		t.Fatalf("query should not fail when hydration fails: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 finding even without hydration, got %d", len(results))
	}
}

func TestClientQuery_VulnAPIHTTPError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/querybatch", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"vulns":[{"id":"GHSA-500"}]}]}`))
	})
	mux.HandleFunc("/vulns/GHSA-500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"message":"gateway error"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClientWithEndpoint(server.URL+"/querybatch", server.URL+"/vulns", time.Hour)
	deps := []shared.Dependency{{Name: "pkg", Version: "1.0.0", Ecosystem: "npm"}}
	// Query should not error even if vuln API returns 500 (hydration is best-effort)
	results, err := client.Query(context.Background(), deps)
	if err != nil {
		t.Fatalf("hydration failure should not break query: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 finding, got %d", len(results))
	}
}

func TestFixedVersionForRange_SemverFixFound(t *testing.T) {
	t.Parallel()
	r := osvRange{
		Type: "SEMVER",
		Events: []map[string]string{
			{"introduced": "0"},
			{"fixed": "1.2.4"},
		},
	}
	result := fixedVersionForRange(r, "1.2.3")
	if result != "1.2.4" {
		t.Errorf("expected fix version 1.2.4, got %q", result)
	}
}

func TestFixedVersionForRange_AlreadyPatched(t *testing.T) {
	t.Parallel()
	r := osvRange{
		Type: "SEMVER",
		Events: []map[string]string{
			{"introduced": "0"},
			{"fixed": "1.2.0"},
		},
	}
	result := fixedVersionForRange(r, "1.3.0")
	if result != "" {
		t.Errorf("expected no fix version for already-patched package, got %q", result)
	}
}

func TestFixedVersionForRange_NonSemverIgnored(t *testing.T) {
	t.Parallel()
	r := osvRange{
		Type: "GIT",
		Events: []map[string]string{
			{"introduced": "0"},
			{"fixed": "1.2.4"},
		},
	}
	result := fixedVersionForRange(r, "1.2.3")
	if result != "" {
		t.Errorf("expected no fix version for non-SEMVER range, got %q", result)
	}
}

func TestFixedVersionForRange_LastAffected(t *testing.T) {
	t.Parallel()
	r := osvRange{
		Type: "SEMVER",
		Events: []map[string]string{
			{"introduced": "0"},
			{"last_affected": "1.2.3"},
		},
	}
	result := fixedVersionForRange(r, "1.2.3")
	if result != "" {
		t.Errorf("expected no fix version when current is last_affected, got %q", result)
	}
}

func TestEarliestFixedVersion(t *testing.T) {
	t.Parallel()
	events := []map[string]string{
		{"fixed": "2.0.0"},
		{"fixed": "1.5.0"},
		{"fixed": "1.8.0"},
	}
	result := earliestFixedVersion(events)
	if result != "1.5.0" {
		t.Errorf("expected earliest fix version 1.5.0, got %q", result)
	}
}

func TestEarliestFixedVersion_NoFixed(t *testing.T) {
	t.Parallel()
	events := []map[string]string{
		{"introduced": "0"},
		{"last_affected": "1.0.0"},
	}
	result := earliestFixedVersion(events)
	if result != "" {
		t.Errorf("expected no fix version, got %q", result)
	}
}

func TestSummarizeVulnerability_UsesSummary(t *testing.T) {
	t.Parallel()
	result := summarizeVulnerability("Remote code execution", "Details about RCE", "express")
	if result != "Remote code execution" {
		t.Errorf("expected summary, got %q", result)
	}
}

func TestSummarizeVulnerability_FallbackToDetails(t *testing.T) {
	t.Parallel()
	result := summarizeVulnerability("", "A critical vulnerability was found in the HTTP parser.", "http-parser")
	if result != "A critical vulnerability was found in the HTTP parser." {
		t.Errorf("expected details, got %q", result)
	}
}

func TestSummarizeVulnerability_GenericSummarySkipped(t *testing.T) {
	t.Parallel()
	tests := []string{
		"Vulnerability detected",
		"Security vulnerability",
		"Known vulnerability",
		"Vulnerability detected in express",
		"Vulnerability in express",
		"Security vulnerability in express",
	}
	for _, summary := range tests {
		result := summarizeVulnerability(summary, "More details here", "express")
		if result != "More details here" {
			t.Errorf("for summary %q expected details, got %q", summary, result)
		}
	}
}

func TestSummarizeVulnerability_FallbackToPackageName(t *testing.T) {
	t.Parallel()
	result := summarizeVulnerability("", "", "axios")
	if result != "Vulnerability detected in axios" {
		t.Errorf("expected package fallback, got %q", result)
	}
}

func TestSummarizeVulnerability_CompleteEmpty(t *testing.T) {
	t.Parallel()
	result := summarizeVulnerability("", "", "")
	if result != "Vulnerability detected" {
		t.Errorf("expected default message, got %q", result)
	}
}

func TestFirstAdvisorySentence_ExtractsFirstSentence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"First sentence. Second sentence.", "First sentence."},
		{"Alert! Something happened.", "Alert!"},
		{"Question? More text.", "Question?"},
		{"No punctuation here", "No punctuation here"},
		{"", ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := firstAdvisorySentence(tt.input)
			if result != tt.expected {
				t.Errorf("firstAdvisorySentence(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCompactWhitespace(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello   world  ", "hello world"},
		{"no-extra", "no-extra"},
		{"\ttabs\tand\nnewlines", "tabs and newlines"},
	}
	for _, tt := range tests {
		result := compactWhitespace(tt.input)
		if result != tt.expected {
			t.Errorf("compactWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsGenericVulnerabilitySummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		summary     string
		packageName string
		expected    bool
	}{
		{"Vulnerability detected", "", true},
		{"Security vulnerability", "", true},
		{"Known vulnerability", "", true},
		{"Vulnerability detected in lodash", "lodash", true},
		{"Vulnerability in lodash", "lodash", true},
		{"Remote code execution", "express", false},
		{"", "express", true}, // Empty summary is generic
	}
	for _, tt := range tests {
		t.Run(tt.summary, func(t *testing.T) {
			result := isGenericVulnerabilitySummary(tt.summary, tt.packageName)
			if result != tt.expected {
				t.Errorf("isGeneric(%q, %q) = %v, want %v", tt.summary, tt.packageName, result, tt.expected)
			}
		})
	}
}

func TestAdvisoryNeedsHydration(t *testing.T) {
	t.Parallel()
	needsHydration := advisoryNeedsHydration(osvAdvisory{ID: "GHSA-test"})
	if !needsHydration {
		t.Error("expected advisory with no affected packages to need hydration")
	}
	doesNotNeed := advisoryNeedsHydration(osvAdvisory{
		ID:       "GHSA-test",
		Affected: []osvAffected{{Package: struct {
			Name string `json:"name"`
		}{Name: "pkg"}}},
	})
	if doesNotNeed {
		t.Error("expected advisory with affected packages to not need hydration")
	}
}
