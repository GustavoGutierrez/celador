package osv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestClientQueryFallsBackToDetailsWhenSummaryIsBlank(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/querybatch" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"results": [{
				"vulns": [{
					"id": "GHSA-abcd-1234",
					"summary": "",
					"details": "Prototype pollution in lodash allows crafted input to modify object prototypes.",
					"affected": [{"package": {"name": "lodash"}, "ranges": [{"type": "SEMVER", "events": [{"introduced": "0"}, {"fixed": "4.17.21"}]}]}]
				}]
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(24 * time.Hour)
	client.endpoint = server.URL + "/v1/querybatch"
	client.vulnAPI = server.URL + "/v1/vulns"

	findings, err := client.Query(context.Background(), []shared.Dependency{{
		Name:      "lodash",
		Version:   "4.17.20",
		Ecosystem: "npm",
	}})
	if err != nil {
		t.Fatalf("query osv: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	if findings[0].Summary != "Prototype pollution in lodash allows crafted input to modify object prototypes." {
		t.Fatalf("expected details fallback summary, got %q", findings[0].Summary)
	}
	if findings[0].FixVersion != "4.17.21" {
		t.Fatalf("expected fix version, got %q", findings[0].FixVersion)
	}
}

func TestClientQueryHydratesMinimalBatchResponses(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/querybatch":
			_, _ = w.Write([]byte(`{
				"results": [{
					"vulns": [{
						"id": "GHSA-f23m-r3pf-42rh"
					}]
				}]
			}`))
		case "/v1/vulns/GHSA-f23m-r3pf-42rh":
			_, _ = w.Write([]byte(`{
				"id": "GHSA-f23m-r3pf-42rh",
				"summary": "lodash vulnerable to Prototype Pollution via array path bypass in _.unset and _.omit",
				"details": "Patched in 4.18.0.",
				"affected": [{
					"package": {"name": "lodash"},
					"ranges": [{"type": "SEMVER", "events": [{"introduced": "0"}, {"fixed": "4.18.0"}]}]
				}]
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(24 * time.Hour)
	client.endpoint = server.URL + "/v1/querybatch"
	client.vulnAPI = server.URL + "/v1/vulns"

	findings, err := client.Query(context.Background(), []shared.Dependency{{
		Name:      "lodash",
		Version:   "4.17.20",
		Ecosystem: "npm",
	}})
	if err != nil {
		t.Fatalf("query osv: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	if findings[0].FixVersion != "4.18.0" || !findings[0].Fixable {
		t.Fatalf("expected hydrated fix version, got %+v", findings[0])
	}
	if findings[0].Summary == "Vulnerability detected in lodash" {
		t.Fatalf("expected hydrated summary, got %+v", findings[0])
	}
}

func TestFixedVersionForPackageSelectsApplicableRange(t *testing.T) {
	t.Parallel()

	vuln := osvAdvisory{
		Affected: []osvAffected{{
			Ranges: []osvRange{
				{Type: "SEMVER", Events: []map[string]string{{"introduced": "10.0.0"}, {"fixed": "15.5.14"}}},
				{Type: "SEMVER", Events: []map[string]string{{"introduced": "16.0.0"}, {"fixed": "16.1.7"}}},
			},
		}},
	}
	vuln.Affected[0].Package.Name = "next"

	if got := fixedVersionForPackage(vuln, "next", "15.2.4"); got != "15.5.14" {
		t.Fatalf("expected 15.5.14 for 15.x branch, got %q", got)
	}
	if got := fixedVersionForPackage(vuln, "next", "16.0.5"); got != "16.1.7" {
		t.Fatalf("expected 16.1.7 for 16.x branch, got %q", got)
	}
}

func TestSummarizeVulnerabilityPrefersDetailsWhenSummaryIsGeneric(t *testing.T) {
	t.Parallel()

	got := summarizeVulnerability(
		"Vulnerability in lodash",
		"Prototype pollution in lodash allows crafted input to modify object prototypes. Additional metadata follows.",
		"lodash",
	)
	if got != "Prototype pollution in lodash allows crafted input to modify object prototypes." {
		t.Fatalf("expected detailed advisory sentence, got %q", got)
	}
}

func TestSummarizeVulnerabilityKeepsSpecificSummary(t *testing.T) {
	t.Parallel()

	got := summarizeVulnerability(
		"Prototype pollution in lodash merge helper",
		"Prototype pollution in lodash allows crafted input to modify object prototypes.",
		"lodash",
	)
	if got != "Prototype pollution in lodash merge helper" {
		t.Fatalf("expected specific summary to win, got %q", got)
	}
}
