package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/audit"
	"github.com/GustavoGutierrez/celador/internal/core/fix"
	"github.com/GustavoGutierrez/celador/internal/core/install"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
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

func TestInstallCommandAllowsCleanPreflightInCI(t *testing.T) {
	t.Parallel()
	pm := &helpers.StubPM{}
	ui := &helpers.StubUI{}
	rt := &Runtime{
		Root: fsadapter.NewOSFileSystem(t.TempDir()).ExecRoot(),
		TTY:  false,
		CI:   true,
		UI:   ui,
		InstallSv: install.NewService(
			helpers.StubDetector{Workspace: shared.Workspace{Root: "/tmp/project", PackageManager: shared.PackageManagerNPM}},
			helpers.StubMetadata{Assessment: shared.InstallAssessment{Package: "left-pad", Risk: shared.SeverityLow, ShouldPrompt: false}},
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
