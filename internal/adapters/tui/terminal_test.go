package tui

import (
	"bytes"
	"context"
	"encoding/json"
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
	}, shared.ScanRenderOptions{})
	if err != nil {
		t.Fatalf("render scan: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "High findings:") {
		t.Fatalf("expected severity heading, got %q", got)
	}
	if !strings.Contains(got, "GHSA-abcd-1234: package lodash: Vulnerability detected in lodash: fixed in 4.17.21") {
		t.Fatalf("expected formatted OSV finding, got %q", got)
	}
	if strings.Contains(got, "GHSA-abcd-1234:\n") {
		t.Fatalf("expected non-empty summary in output, got %q", got)
	}
}

func TestRenderInitUsesChecklistStyleSections(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderInit(context.Background(), shared.InitReport{
		Title:    "Initialized /tmp/demo (pnpm)",
		Subtitle: "Workspace hardening completed with conservative defaults.",
		Sections: []shared.ChecklistSection{
			{
				Title:   "Detecting package manager",
				Summary: "Found pnpm via pnpm-lock.yaml",
				Items: []shared.ChecklistItem{{
					Label:  "lockfile",
					Value:  "pnpm-lock.yaml present",
					Status: shared.ChecklistStatusUnchanged,
					Detail: "Celador will use the root lockfile as the source of truth for dependency scanning.",
				}},
			},
			{
				Title:   "Securing .npmrc",
				Summary: "new file",
				Items: []shared.ChecklistItem{{
					Label:  "ignore-scripts",
					Value:  "true",
					Status: shared.ChecklistStatusNew,
					Detail: "Blocks dependency install scripts unless they are explicitly approved.",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("render init: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Initialized /tmp/demo (pnpm)",
		"Detecting package manager",
		"Found pnpm via pnpm-lock.yaml",
		"✓ lockfile = pnpm-lock.yaml present OK",
		"Securing .npmrc",
		"✓ ignore-scripts = true NEW",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestRenderBrandingHeaderShowsBannerSloganAndVersion(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderBrandingHeader(context.Background(), "v1.2.3")
	if err != nil {
		t.Fatalf("render branding header: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		shared.CeladorBranding.ASCIIArt[0],
		shared.CeladorBranding.ASCIIArt[len(shared.CeladorBranding.ASCIIArt)-1],
		shared.CeladorBranding.Slogan,
		"Version: v1.2.3",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
	if !strings.HasPrefix(got, shared.CeladorBranding.ASCIIArt[0]) {
		t.Fatalf("expected banner at start of output, got %q", got)
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
	}, shared.ScanRenderOptions{})
	if err != nil {
		t.Fatalf("render scan: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "tailwind-dynamic-arbitrary-value: target src/app.tsx:12: Tailwind arbitrary value uses string interpolation") {
		t.Fatalf("expected readable rule finding, got %q", got)
	}
}

func TestRenderScanDeduplicatesIdenticalRenderedFindings(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	finding := shared.Finding{
		ID:          "GHSA-abcd-1234",
		Source:      shared.FindingSourceOSV,
		Severity:    shared.SeverityHigh,
		PackageName: "@smithy/config-resolver",
		Target:      "@smithy/config-resolver",
		Summary:     "Prototype pollution in config resolution",
		FixVersion:  "3.0.7",
		Fixable:     true,
	}

	err := ui.RenderScan(context.Background(), shared.ScanResult{
		Fingerprint: "fp-dup",
		Findings:    []shared.Finding{finding, finding},
	}, shared.ScanRenderOptions{})
	if err != nil {
		t.Fatalf("render scan: %v", err)
	}

	got := out.String()
	wantLine := "- [high] GHSA-abcd-1234: package @smithy/config-resolver: Prototype pollution in config resolution: fixed in 3.0.7"
	if strings.Count(got, wantLine) != 1 {
		t.Fatalf("expected deduplicated finding line once, got %q", got)
	}
	if !strings.Contains(got, "Findings: 1 (ignored: 0)") {
		t.Fatalf("expected rendered findings count to reflect deduplication, got %q", got)
	}
	if strings.Count(got, "GHSA-abcd-1234") != 1 {
		t.Fatalf("expected duplicate GHSA entry to be removed, got %q", got)
	}
}

func TestRenderScanSurfacesDistinctTargetContextForDuplicateOSVFindings(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderScan(context.Background(), shared.ScanResult{
		Fingerprint: "fp-context",
		Findings: []shared.Finding{
			{
				ID:          "GHSA-abcd-1234",
				Source:      shared.FindingSourceOSV,
				Severity:    shared.SeverityHigh,
				PackageName: "lodash",
				Target:      "apps/api/package-lock.json",
				Summary:     "Prototype pollution in merge helper",
			},
			{
				ID:          "GHSA-abcd-1234",
				Source:      shared.FindingSourceOSV,
				Severity:    shared.SeverityHigh,
				PackageName: "lodash",
				Target:      "apps/web/package-lock.json",
				Summary:     "Prototype pollution in merge helper",
			},
		},
	}, shared.ScanRenderOptions{})
	if err != nil {
		t.Fatalf("render scan: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"High findings:",
		"- [high] GHSA-abcd-1234: package lodash (target apps/api/package-lock.json): Prototype pollution in merge helper",
		"- [high] GHSA-abcd-1234: package lodash (target apps/web/package-lock.json): Prototype pollution in merge helper",
		"Findings: 2 (ignored: 0)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestRenderScanGroupsFindingsBySeverity(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderScan(context.Background(), shared.ScanResult{
		Fingerprint: "fp-grouped",
		Findings: []shared.Finding{
			{ID: "GHSA-critical", Source: shared.FindingSourceOSV, Severity: shared.SeverityCritical, PackageName: "axios", Target: "axios", Summary: "Remote code execution in redirect handling"},
			{ID: "rule-medium", Source: shared.FindingSourceRule, Severity: shared.SeverityMedium, Target: "src/app.tsx", Summary: "Review dynamic class composition"},
		},
	}, shared.ScanRenderOptions{})
	if err != nil {
		t.Fatalf("render scan: %v", err)
	}

	got := out.String()
	criticalIndex := strings.Index(got, "Critical findings:")
	mediumIndex := strings.Index(got, "Medium findings:")
	if criticalIndex < 0 || mediumIndex < 0 {
		t.Fatalf("expected grouped severity headings, got %q", got)
	}
	if criticalIndex > mediumIndex {
		t.Fatalf("expected critical findings before medium findings, got %q", got)
	}
}

func TestRenderScanVerboseShowsExtraMetadata(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderScan(context.Background(), shared.ScanResult{
		Fingerprint:  "fp-verbose",
		Dependencies: []shared.Dependency{{Name: "lodash"}},
		Workspace:    shared.Workspace{PackageManager: shared.PackageManagerNPM},
		RuleVersion:  "rules-v1",
	}, shared.ScanRenderOptions{Verbose: true})
	if err != nil {
		t.Fatalf("render scan: %v", err)
	}

	got := out.String()
	for _, want := range []string{"Dependencies scanned: 1", "Package manager: npm", "Rule pack: rules-v1", "Status: no actionable findings"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestRenderScanJSONIncludesStructuredScanDetails(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderScan(context.Background(), shared.ScanResult{
		Fingerprint:     "fp-json",
		Dependencies:    []shared.Dependency{{Name: "lodash", Version: "4.17.20"}},
		IgnoredCount:    1,
		FromCache:       true,
		OfflineFallback: true,
		RuleVersion:     "rules-v1",
		Workspace:       shared.Workspace{Root: "/tmp/project", PackageManager: shared.PackageManagerNPM, Lockfiles: []string{"/tmp/project/package-lock.json"}},
		Findings: []shared.Finding{{
			ID:          "GHSA-abcd-1234",
			Source:      shared.FindingSourceOSV,
			Severity:    shared.SeverityHigh,
			PackageName: "lodash",
			Target:      "lodash",
			Summary:     "Prototype pollution in merge helper",
			FixVersion:  "4.17.21",
			Fixable:     true,
		}},
	}, shared.ScanRenderOptions{Format: shared.ScanRenderFormatJSON})
	if err != nil {
		t.Fatalf("render scan json: %v", err)
	}

	var payload struct {
		Fingerprint          string `json:"fingerprint"`
		RenderedFindingCount int    `json:"rendered_finding_count"`
		IgnoredCount         int    `json:"ignored_count"`
		RuleVersion          string `json:"rule_version"`
		Cache                struct {
			FromCache       bool `json:"from_cache"`
			OfflineFallback bool `json:"offline_fallback"`
		} `json:"cache"`
		Findings []struct {
			ID         string `json:"id"`
			FixVersion string `json:"fix_version"`
			Rendered   string `json:"rendered"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal json output: %v\n%s", err, out.String())
	}
	if payload.Fingerprint != "fp-json" || payload.RenderedFindingCount != 1 || payload.IgnoredCount != 1 || payload.RuleVersion != "rules-v1" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if !payload.Cache.FromCache || !payload.Cache.OfflineFallback {
		t.Fatalf("expected cache flags in payload: %+v", payload.Cache)
	}
	if len(payload.Findings) != 1 || payload.Findings[0].FixVersion != "4.17.21" || payload.Findings[0].ID != "GHSA-abcd-1234" {
		t.Fatalf("unexpected findings payload: %+v", payload.Findings)
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
		"Remediation analysis complete",
		"Plan summary",
		"✓ safe operations = 0 OK",
		"Why nothing was planned",
		"✓ no known fixed version = 1 finding INFO",
		"Examples: OSV-1 — package lodash",
		"✓ manual remediation required = 1 finding INFO",
		"Examples: rule-1 — target src/app.tsx",
		"✓ outside current remediation scope = 1 finding INFO",
		"Examples: OSV-2 — target lockfile-only",
		"Diff preview",
		"✓ package.json diff = no changes INFO",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestRenderFixPlanShowsSectionedOperationsAndDiff(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderFixPlan(context.Background(), shared.FixPlan{
		Summary: "Planned 2 conservative remediation operations",
		Operations: []shared.FixOperation{
			{File: "/tmp/demo/package.json", ManifestSection: "dependencies", PackageName: "lodash", CurrentVersion: "4.17.20", ProposedVersion: "4.17.21", Strategy: "bump", RequiresInstall: true},
			{File: "/tmp/demo/package.json", PackageName: "minimist", ProposedVersion: "1.2.8", Strategy: "override", RequiresInstall: true},
		},
		DryRunDiff: "--- package.json\n+++ package.json\n@@ -1,3 +1,3 @@\n",
	})
	if err != nil {
		t.Fatalf("render fix plan: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Remediation plan ready",
		"Planned operations",
		"✓ lodash = dependencies 4.17.20 -> 4.17.21 UPDATED",
		"bump lodash in dependencies from 4.17.20 to 4.17.21",
		"✓ minimist = override -> 1.2.8 NEW",
		"override minimist to 1.2.8 in package.json",
		"Follow-up: reinstall dependencies to refresh the lockfile.",
		"Diff preview",
		"--- package.json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestRenderInstallAssessmentUsesSectionedPreflightReport(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderInstallAssessment(context.Background(), shared.InstallAssessment{
		Package:       "left-pad",
		Manager:       shared.PackageManagerPNPM,
		Risk:          shared.SeverityHigh,
		Unknown:       true,
		ShouldPrompt:  true,
		Reasons:       []string{"Package publishes install scripts", "Registry metadata could not be fully verified"},
		SuggestedArgs: []string{"pnpm", "add", "--ignore-scripts", "left-pad"},
	})
	if err != nil {
		t.Fatalf("render install assessment: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Install preflight",
		"Assessment complete for left-pad before pnpm install. Approval is recommended before continuing.",
		"Package summary",
		"✓ package = left-pad OK",
		"✓ package manager = pnpm OK",
		"✓ risk = High INFO",
		"✓ registry status = unknown INFO",
		"Risk review",
		"Celador recommends an explicit approval before package manager execution.",
		"✓ Package publishes install scripts INFO",
		"✓ Registry metadata could not be fully verified INFO",
		"Suggested command",
		"✓ command = pnpm add --ignore-scripts left-pad INFO",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestRenderInstallTimelineReportsRealExecutionStages(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ui := NewTerminalUI(strings.NewReader(""), &out, false, true)

	err := ui.RenderInstallTimeline(context.Background(), shared.InstallTimeline{
		Assessment: shared.InstallAssessment{
			Package:       "left-pad",
			Manager:       shared.PackageManagerPNPM,
			Risk:          shared.SeverityHigh,
			ShouldPrompt:  true,
			SuggestedArgs: []string{"pnpm", "add", "--ignore-scripts", "left-pad"},
		},
		RequestedArgs:  []string{"left-pad"},
		Command:        []string{"pnpm", "install", "left-pad"},
		Approval:       shared.InstallApprovalPromptApproved,
		ExecutionState: shared.InstallExecutionSucceeded,
	})
	if err != nil {
		t.Fatalf("render install timeline: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Install timeline",
		"Execution finished for left-pad with pnpm.",
		"1. Request and preflight",
		"✓ requested packages = left-pad OK",
		"✓ safer suggested command = pnpm add --ignore-scripts left-pad INFO",
		"2. Approval decision",
		"✓ approval status = granted interactively INFO",
		"3. Package manager execution",
		"✓ command = pnpm install left-pad UPDATED",
		"State: completed",
		"4. Security summary",
		"✓ post-install security check = not run automatically INFO",
		"Run `celador scan` to audit the updated lockfiles explicitly.",
		"5. Final outcome",
		"The install finished and approval was required and granted interactively before execution.",
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
