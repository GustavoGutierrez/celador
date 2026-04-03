package tui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestRenderScanFormatsOSVFindingWithContextAndFixVersion(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderScan(context.Background(), shared.ScanResult{
		Fingerprint: "fp-123",
		Findings: []shared.Finding{{
			ID:          "GHSA-abcd-1234",
			Source:      shared.FindingSourceOSV,
			Severity:    shared.SeverityHigh,
			PackageName: "lodash",
			Target:      "lodash",
			Summary:     "",
			FixVersion:  "4.17.21",
			Fixable:     true,
		}},
	})
	if err != nil {
		t.Fatalf("render scan: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "GHSA-abcd-1234: package lodash: Vulnerability detected in lodash: fixed in 4.17.21") {
		t.Fatalf("expected formatted OSV finding, got %q", got)
	}
	if strings.Contains(got, "GHSA-abcd-1234:\n") {
		t.Fatalf("expected non-empty summary in output, got %q", got)
	}
}

func TestRenderScanFormatsRuleFindingWithLocation(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderScan(context.Background(), shared.ScanResult{
		Fingerprint: "fp-456",
		Findings: []shared.Finding{{
			ID:       "tailwind-dynamic-arbitrary-value",
			Source:   shared.FindingSourceRule,
			Severity: shared.SeverityHigh,
			Target:   "src/app.tsx",
			Summary:  "Tailwind arbitrary value uses string interpolation",
			Locations: []shared.FindingLocation{{
				Path: "src/app.tsx",
				Line: 12,
			}},
		}},
	})
	if err != nil {
		t.Fatalf("render scan: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "tailwind-dynamic-arbitrary-value: target src/app.tsx:12: Tailwind arbitrary value uses string interpolation") {
		t.Fatalf("expected readable rule finding, got %q", got)
	}
}
