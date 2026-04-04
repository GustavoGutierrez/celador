package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	uiadapter "github.com/GustavoGutierrez/celador/internal/adapters/tui"
	"github.com/GustavoGutierrez/celador/internal/core/audit"
	"github.com/GustavoGutierrez/celador/internal/core/fix"
	"github.com/GustavoGutierrez/celador/internal/core/install"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	versioncore "github.com/GustavoGutierrez/celador/internal/core/version"
	"github.com/GustavoGutierrez/celador/internal/core/workspace"
	"github.com/GustavoGutierrez/celador/internal/ports"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestRootNoInteractiveDisablesPrompting(t *testing.T) {
	t.Parallel()
	rt, ui, patchWriter := newFixRuntime(t, []shared.Finding{{ID: "OSV-1", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "test", FixVersion: "4.17.21", Fixable: true}})
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"--no-interactive", "fix"})

	err := cmd.ExecuteContext(context.Background())
	assertExitCode(t, err, 2)
	if ui.ConfirmCalls != 0 {
		t.Fatalf("expected no prompt in no-interactive mode")
	}
	if len(patchWriter.Applied.Operations) != 0 {
		t.Fatalf("expected no changes to be applied without --yes")
	}
}

func TestScanCommandUnsupportedInput(t *testing.T) {
	t.Parallel()
	fs := fsadapter.NewOSFileSystem(t.TempDir())
	rt := &Runtime{
		Root:    fs.ExecRoot(),
		TTY:     false,
		CI:      true,
		FS:      fs,
		UI:      &helpers.StubUI{},
		ScanSvc: audit.NewService(helpers.StubDetector{Workspace: shared.Workspace{Root: fs.ExecRoot(), PackageManager: shared.PackageManagerNPM}}, helpers.StubIgnore{}, helpers.StubRuleLoader{Version: "v1"}, helpers.StubRuleEvaluator{}, &helpers.StubOSV{}, &helpers.StubCache{}, helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)}, 24*time.Hour, nil),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"scan"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil || err.Error() != "no supported lockfile found (package-lock.json, pnpm-lock.yaml, bun.lock, deno.lock)" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanCommandUsesRenderedFindingCountInExitMessage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWriteAppFile(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"},"dependencies":{"@smithy/config-resolver":"3.0.6"}}`)
	mustWriteAppFile(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/@smithy/config-resolver":{"version":"3.0.6"}}}`)

	var out bytes.Buffer
	ui := uiadapter.NewTerminalUI(strings.NewReader(""), &out, false, true)
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
	scanSvc := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Findings: []shared.Finding{finding, finding}},
		&helpers.StubCache{},
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{audit.NewNPMParser(fs)},
	)
	rt := &Runtime{
		Root:      root,
		TTY:       false,
		CI:        true,
		FS:        fs,
		UI:        ui,
		VersionSv: versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", nil),
		ScanSvc:   scanSvc,
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"scan"})

	err := cmd.ExecuteContext(context.Background())
	assertExitCode(t, err, 3)
	if err.Error() != "1 findings detected" {
		t.Fatalf("expected deduplicated exit message, got %q", err.Error())
	}
	got := out.String()
	if !strings.HasPrefix(got, shared.CeladorBranding.ASCIIArt[0]) {
		t.Fatalf("expected branding header at start of scan output, got %q", got)
	}
	if !strings.Contains(got, "Version: v1.2.3") {
		t.Fatalf("expected branding version in scan output, got %q", got)
	}
	if !strings.Contains(got, "Findings: 1 (ignored: 0)") {
		t.Fatalf("expected rendered header to use the same finding count, got %q", got)
	}
	if !strings.Contains(got, "High findings:") {
		t.Fatalf("expected grouped scan output, got %q", got)
	}
	if strings.Count(got, "GHSA-abcd-1234") != 1 {
		t.Fatalf("expected a single rendered finding, got %q", got)
	}
}

func TestScanCommandJSONFlagRendersStructuredOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWriteAppFile(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"},"dependencies":{"lodash":"4.17.20"}}`)
	mustWriteAppFile(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)

	var out bytes.Buffer
	ui := uiadapter.NewTerminalUI(strings.NewReader(""), &out, false, true)
	finding := shared.Finding{ID: "GHSA-abcd-1234", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "Prototype pollution in merge helper", FixVersion: "4.17.21", Fixable: true}
	scanSvc := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Findings: []shared.Finding{finding, finding}},
		&helpers.StubCache{},
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{audit.NewNPMParser(fs)},
	)
	rt := &Runtime{Root: root, TTY: false, CI: true, FS: fs, UI: ui, VersionSv: versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", nil), ScanSvc: scanSvc}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"scan", "--json"})

	err := cmd.ExecuteContext(context.Background())
	assertExitCode(t, err, 3)
	if err.Error() != "1 findings detected" {
		t.Fatalf("expected deduplicated exit message, got %q", err.Error())
	}

	var payload struct {
		RenderedFindingCount int `json:"rendered_finding_count"`
		RawFindingCount      int `json:"raw_finding_count"`
		Findings             []struct {
			ID         string `json:"id"`
			FixVersion string `json:"fix_version"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal scan json: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), shared.CeladorBranding.Slogan) || strings.Contains(out.String(), "Version: v1.2.3") {
		t.Fatalf("expected scan json output without branding header, got %q", out.String())
	}
	if payload.RenderedFindingCount != 1 || payload.RawFindingCount != 2 {
		t.Fatalf("unexpected rendered/raw counts: %+v", payload)
	}
	if len(payload.Findings) != 2 || payload.Findings[0].ID != "GHSA-abcd-1234" || payload.Findings[0].FixVersion != "4.17.21" {
		t.Fatalf("unexpected findings payload: %+v", payload.Findings)
	}
}

func TestScanCommandVerboseFlagPassesVerboseRenderOption(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWriteAppFile(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"},"dependencies":{"lodash":"4.17.20"}}`)
	mustWriteAppFile(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)

	ui := &helpers.StubUI{}
	scanSvc := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{},
		&helpers.StubCache{},
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{audit.NewNPMParser(fs)},
	)
	rt := &Runtime{Root: root, TTY: false, CI: true, FS: fs, UI: ui, ScanSvc: scanSvc}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"scan", "--verbose"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("scan should succeed: %v", err)
	}
	if ui.ScanCalls != 1 {
		t.Fatalf("expected scan render call, got %d", ui.ScanCalls)
	}
	if !ui.LastScanOpts.Verbose || ui.LastScanOpts.Format != shared.ScanRenderFormatText {
		t.Fatalf("expected verbose text render options, got %+v", ui.LastScanOpts)
	}
}

func TestScanCommandRendersBrandingHeaderInTextMode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWriteAppFile(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"},"dependencies":{"lodash":"4.17.20"}}`)
	mustWriteAppFile(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)

	ui := &helpers.StubUI{}
	scanSvc := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{},
		&helpers.StubCache{},
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{audit.NewNPMParser(fs)},
	)
	rt := &Runtime{Root: root, TTY: false, CI: true, FS: fs, UI: ui, VersionSv: versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", nil), ScanSvc: scanSvc}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"scan"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("scan should succeed: %v", err)
	}
	if ui.BrandingCalls != 1 {
		t.Fatalf("expected branding header once, got %d", ui.BrandingCalls)
	}
	if ui.LastVersion != "v1.2.3" {
		t.Fatalf("expected branding version v1.2.3, got %q", ui.LastVersion)
	}
	if ui.ScanCalls != 1 {
		t.Fatalf("expected scan render once, got %d", ui.ScanCalls)
	}
	got := ui.Output.String()
	if !strings.HasPrefix(got, "[branding v1.2.3]\n") {
		t.Fatalf("expected branding header first, got %q", got)
	}
}

func TestFixCommandNoSafeRemediationReturnsExitCodeFour(t *testing.T) {
	t.Parallel()
	rt, _, patchWriter := newFixRuntime(t, nil)
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"fix"})

	err := cmd.ExecuteContext(context.Background())
	assertExitCode(t, err, 4)
	if len(patchWriter.Applied.Operations) != 0 {
		t.Fatalf("expected no patch application when no safe fix exists")
	}
}

func TestFixCommandRendersBrandingHeaderBeforePlan(t *testing.T) {
	t.Parallel()

	rt, ui, _ := newFixRuntime(t, []shared.Finding{{
		ID:          "OSV-1",
		Source:      shared.FindingSourceOSV,
		Severity:    shared.SeverityHigh,
		PackageName: "lodash",
		Target:      "lodash",
		Summary:     "Prototype pollution in merge helper",
		FixVersion:  "4.17.21",
		Fixable:     true,
	}})
	rt.VersionSv = versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", nil)
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"fix", "--diff"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("fix --diff should succeed: %v", err)
	}
	if ui.BrandingCalls != 1 {
		t.Fatalf("expected branding header once, got %d", ui.BrandingCalls)
	}
	if ui.LastVersion != "v1.2.3" {
		t.Fatalf("expected branding version v1.2.3, got %q", ui.LastVersion)
	}
	got := ui.Output.String()
	if !strings.HasPrefix(got, "[branding v1.2.3]\n") {
		t.Fatalf("expected branding header first, got %q", got)
	}
	if strings.Count(got, "[branding v1.2.3]") != 1 {
		t.Fatalf("expected branding header only once, got %q", got)
	}
	if !strings.Contains(got, "lodash (dependencies)@4.17.20") {
		t.Fatalf("expected fix plan output after branding header, got %q", got)
	}
	if ui.ConfirmCalls != 0 {
		t.Fatalf("expected no confirmation prompt for --diff mode")
	}
}

func TestVersionFlagPrintsCurrentVersionAndHomebrewUpgradeHint(t *testing.T) {
	t.Parallel()

	ui := &helpers.StubUI{}
	rt := &Runtime{
		UI:        ui,
		VersionSv: versionServiceForTest("v1.2.3", "v1.3.0", "/opt/homebrew/Cellar/celador/1.2.3/bin/celador", nil),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("version command should succeed: %v", err)
	}
	got := ui.Output.String()
	for _, want := range []string{
		"celador v1.2.3",
		"Update available: v1.3.0",
		"brew update && brew upgrade GustavoGutierrez/celador/celador",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestVersionFlagStillPrintsCurrentVersionWhenCheckFails(t *testing.T) {
	t.Parallel()

	ui := &helpers.StubUI{}
	rt := &Runtime{
		UI:        ui,
		VersionSv: versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", errors.New("boom")),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("version command should ignore lookup failures: %v", err)
	}
	got := ui.Output.String()
	if !strings.Contains(got, "celador v1.2.3") {
		t.Fatalf("expected current version in output, got %q", got)
	}
	if strings.Contains(got, "Update available:") {
		t.Fatalf("did not expect update banner when lookup fails, got %q", got)
	}
}

func TestInitCommandRendersBrandingHeaderBeforeChecklist(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWriteAppFile(t, fs, filepath.Join(root, "package.json"), `{"name":"demo","version":"1.0.0","engines":{"node":"20.0.0"}}`)
	mustWriteAppFile(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{}}`)

	ui := &helpers.StubUI{}
	rt := &Runtime{
		Root: root,
		TTY:  false,
		CI:   true,
		FS:   fs,
		UI:   ui,
		InitSvc: workspace.NewService(
			fs,
			helpers.StubDetector{Workspace: shared.Workspace{
				Root:           root,
				PackageManager: shared.PackageManagerNPM,
				Lockfiles:      []string{filepath.Join(root, "package-lock.json")},
				ManifestPath:   filepath.Join(root, "package.json"),
				ConfigPath:     filepath.Join(root, ".celador.yaml"),
			}},
			helpers.StubIgnore{},
			ui,
			&helpers.StubNodeVersionDetector{},
		),
		VersionSv: versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", nil),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"init"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("init should succeed: %v", err)
	}
	if ui.BrandingCalls != 1 {
		t.Fatalf("expected branding header once, got %d", ui.BrandingCalls)
	}
	if ui.LastVersion != "v1.2.3" {
		t.Fatalf("expected branding version v1.2.3, got %q", ui.LastVersion)
	}
	if ui.InitCalls != 1 {
		t.Fatalf("expected init render once, got %d", ui.InitCalls)
	}
	got := ui.Output.String()
	if !strings.HasPrefix(got, "[branding v1.2.3]\n") {
		t.Fatalf("expected branding header first, got %q", got)
	}
}

func TestInstallCommandAllowsCleanPreflightInCI(t *testing.T) {
	t.Parallel()
	pm := &helpers.StubPM{}
	ui := &helpers.StubUI{}
	rt := &Runtime{
		Root:      fsadapter.NewOSFileSystem(t.TempDir()).ExecRoot(),
		TTY:       false,
		CI:        true,
		UI:        ui,
		VersionSv: versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", nil),
		InstallSv: install.NewService(
			helpers.StubDetector{Workspace: shared.Workspace{Root: "/tmp/project", PackageManager: shared.PackageManagerNPM}},
			&helpers.StubMetadata{Assessment: shared.InstallAssessment{Package: "left-pad", Risk: shared.SeverityLow, ShouldPrompt: false}},
			pm,
			ui,
		),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"install", "left-pad"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("install should succeed in CI for clean preflight: %v", err)
	}
	if len(pm.Calls) != 1 {
		t.Fatalf("expected package manager execution for clean preflight")
	}
	if ui.ConfirmCalls != 0 {
		t.Fatalf("expected no prompt for clean preflight")
	}
	if ui.BrandingCalls != 1 {
		t.Fatalf("expected branding header once, got %d", ui.BrandingCalls)
	}
	if ui.LastVersion != "v1.2.3" {
		t.Fatalf("expected branding version v1.2.3, got %q", ui.LastVersion)
	}
	if ui.TimelineCalls != 2 {
		t.Fatalf("expected running and completed timeline renders, got %d", ui.TimelineCalls)
	}
	if ui.LastTimeline.ExecutionState != shared.InstallExecutionSucceeded {
		t.Fatalf("expected final timeline success state, got %+v", ui.LastTimeline)
	}
	if got := ui.Output.String(); !strings.HasPrefix(got, "[branding v1.2.3]\n") {
		t.Fatalf("expected branding header first, got %q", got)
	} else if strings.Count(got, "[branding v1.2.3]") != 1 {
		t.Fatalf("expected branding header only once, got %q", got)
	} else if !strings.Contains(got, "npm install left-pad") {
		t.Fatalf("expected rendered install command in timeline output, got %q", got)
	}
}

func TestInstallCommandUsesIntentionalRiskApprovalPrompt(t *testing.T) {
	t.Parallel()

	pm := &helpers.StubPM{}
	ui := &helpers.StubUI{ConfirmResult: true}
	rt := &Runtime{
		Root: fsadapter.NewOSFileSystem(t.TempDir()).ExecRoot(),
		TTY:  true,
		CI:   false,
		UI:   ui,
		InstallSv: install.NewService(
			helpers.StubDetector{Workspace: shared.Workspace{Root: "/tmp/project", PackageManager: shared.PackageManagerNPM}},
			&helpers.StubMetadata{Assessment: shared.InstallAssessment{Package: "left-pad", Risk: shared.SeverityHigh, ShouldPrompt: true}},
			pm,
			ui,
		),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"install", "left-pad"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("install should succeed after approval: %v", err)
	}
	if ui.LastPrompt != "Proceed with installing left-pad after reviewing the preflight warnings?" {
		t.Fatalf("unexpected prompt text: %q", ui.LastPrompt)
	}
	if ui.ConfirmCalls != 1 {
		t.Fatalf("expected one prompt, got %d", ui.ConfirmCalls)
	}
	if len(pm.Calls) != 1 {
		t.Fatalf("expected package manager execution after approval")
	}
	if ui.TimelineCalls != 2 {
		t.Fatalf("expected running and completed timeline renders, got %d", ui.TimelineCalls)
	}
	if ui.LastTimeline.Approval != shared.InstallApprovalPromptApproved {
		t.Fatalf("expected interactive approval in final timeline, got %+v", ui.LastTimeline)
	}
}

func TestInstallCommandRendersFailedTimelineWhenPackageManagerFails(t *testing.T) {
	t.Parallel()

	pm := &helpers.StubPM{Err: errors.New("boom")}
	ui := &helpers.StubUI{}
	rt := &Runtime{
		Root: fsadapter.NewOSFileSystem(t.TempDir()).ExecRoot(),
		TTY:  false,
		CI:   true,
		UI:   ui,
		InstallSv: install.NewService(
			helpers.StubDetector{Workspace: shared.Workspace{Root: "/tmp/project", PackageManager: shared.PackageManagerNPM}},
			&helpers.StubMetadata{Assessment: shared.InstallAssessment{Package: "left-pad", Manager: shared.PackageManagerNPM, Risk: shared.SeverityLow, ShouldPrompt: false}},
			pm,
			ui,
		),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"install", "left-pad"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected package manager error, got %v", err)
	}
	if ui.TimelineCalls != 2 {
		t.Fatalf("expected running and failed timeline renders, got %d", ui.TimelineCalls)
	}
	if ui.LastTimeline.ExecutionState != shared.InstallExecutionFailed {
		t.Fatalf("expected failed timeline state, got %+v", ui.LastTimeline)
	}
	if ui.LastTimeline.Failure != "boom" {
		t.Fatalf("expected failure details in timeline, got %+v", ui.LastTimeline)
	}
}

func TestAboutCommandPrintsDeveloperProfileAndVersion(t *testing.T) {
	t.Parallel()

	ui := &helpers.StubUI{}
	rt := &Runtime{
		UI:        ui,
		VersionSv: versionServiceForTest("v1.2.3", "v1.3.0", "/usr/local/bin/celador", nil),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"about"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("about command should succeed: %v", err)
	}
	got := ui.Output.String()
	for _, want := range []string{
		"Gustavo Gutierrez",
		"https://github.com/GustavoGutierrez",
		"v1.2.3",
		"v1.3.0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
	if ui.OverviewCalls != 1 {
		t.Fatalf("expected overview rendering once, got %d", ui.OverviewCalls)
	}
	if ui.Interactive {
		t.Fatalf("about command should use plain-text mode")
	}
}

func TestTUICommandUsesInteractiveModeWhenTTYAvailable(t *testing.T) {
	t.Parallel()

	ui := &helpers.StubUI{}
	rt := &Runtime{
		TTY:       true,
		CI:        false,
		UI:        ui,
		VersionSv: versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", nil),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"tui"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("tui command should succeed: %v", err)
	}
	if !ui.Interactive {
		t.Fatalf("expected tui command to request interactive mode")
	}
}

func TestTUICommandFallsBackToStaticModeInCI(t *testing.T) {
	t.Parallel()

	ui := &helpers.StubUI{}
	rt := &Runtime{
		TTY:       false,
		CI:        true,
		UI:        ui,
		VersionSv: versionServiceForTest("v1.2.3", "", "/usr/local/bin/celador", nil),
	}
	cmd := newRootCommand(rt)
	cmd.SetArgs([]string{"tui"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("tui command should succeed in CI: %v", err)
	}
	if ui.Interactive {
		t.Fatalf("expected tui command to fall back to static mode in CI")
	}
}

func newFixRuntime(t *testing.T, findings []shared.Finding) (*Runtime, *helpers.StubUI, *helpers.StubPatchWriter) {
	t.Helper()
	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWriteAppFile(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"},"dependencies":{"lodash":"4.17.20"}}`)
	mustWriteAppFile(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)
	ui := &helpers.StubUI{ConfirmResult: true}
	patchWriter := &helpers.StubPatchWriter{}
	scanSvc := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Findings: findings},
		&helpers.StubCache{},
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{audit.NewNPMParser(fs)},
	)
	return &Runtime{
		Root:   root,
		TTY:    true,
		CI:     false,
		FS:     fs,
		UI:     ui,
		FixSvc: fix.NewService(scanSvc, patchWriter, fs, ui),
	}, ui, patchWriter
}

func mustWriteAppFile(t *testing.T, fs interface {
	WriteFile(context.Context, string, []byte) error
}, path string, body string) {
	t.Helper()
	if err := fs.WriteFile(context.Background(), path, []byte(body)); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertExitCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected exit code %d, got nil", want)
	}
	var exitErr ExitCoder
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit error, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != want {
		t.Fatalf("expected exit code %d, got %d", want, exitErr.ExitCode())
	}
}

func versionServiceForTest(current string, latest string, executablePath string, err error) *versioncore.Service {
	return versioncore.NewService(current, stubReleaseSource{latest: latest, err: err}, executablePath)
}

type stubReleaseSource struct {
	latest string
	err    error
}

func (s stubReleaseSource) Latest(context.Context) (string, error) {
	return s.latest, s.err
}
