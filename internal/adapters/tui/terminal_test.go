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

func TestRenderFixPlanExplainsZeroPlanCategories(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderFixPlan(context.Background(), shared.FixPlan{
		Summary:    "Planned 0 conservative remediation operations",
		DryRunDiff: "No package.json diff available because no safe manifest changes were planned.\n",
		Reasons: []shared.FixPlanReason{
			{Category: shared.FixPlanReasonNoFixedVersion, Count: 1, Examples: []string{"OSV-1 — package lodash"}},
			{Category: shared.FixPlanReasonManualChange, Count: 1, Examples: []string{"rule-1 — target src/app.tsx"}},
			{Category: shared.FixPlanReasonOutsideScope, Count: 1, Examples: []string{"OSV-2 — target lockfile-only"}},
		},
	})
	if err != nil {
		t.Fatalf("render fix plan: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Why no safe remediation was planned:",
		"finding has no known fixed version (OSV-1 — package lodash)",
		"finding requires manual changes (rule-1 — target src/app.tsx)",
		"finding is outside the current remediation scope (OSV-2 — target lockfile-only)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestRenderOverviewIncludesDeveloperVersionAndHelp(t *testing.T) {
	t.Parallel()

	got := renderOverview(shared.Overview{
		Title:           "Celador CLI",
		Subtitle:        "Zero-trust dependency security.",
		Developer:       "Gustavo Gutierrez",
		GitHubProfile:   "https://github.com/GustavoGutierrez",
		CurrentVersion:  "v1.2.3",
		LatestVersion:   "v1.3.0",
		UpdateAvailable: true,
		UpgradeCommand:  "brew update && brew upgrade GustavoGutierrez/celador/celador",
		Commands:        []shared.OverviewCommand{{Name: "celador scan", Summary: "Audit dependencies.", Example: "celador scan"}},
		QuickStart:      []string{"Run `celador init` once per workspace."},
		Documentation:   []string{"README.md", "docs/commands.md"},
	}, 110, false)

	for _, want := range []string{
		"Gustavo Gutierrez",
		"https://github.com/GustavoGutierrez",
		"Current version: v1.2.3",
		"Latest release: v1.3.0 (update available)",
		"Upgrade guidance: brew update && brew upgrade",
		"GustavoGutierrez/celador/celador",
		"Run `celador about` for a plain-text overview.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}
