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
					"affected": [{"ranges": [{"events": [{"fixed": "4.17.21"}]}]}]
				}]
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(24 * time.Hour)
	client.endpoint = server.URL + "/v1/querybatch"

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
