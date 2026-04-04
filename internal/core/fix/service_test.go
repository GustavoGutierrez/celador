package fix

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/audit"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestPlanCreatesConservativeOps(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	if err := fs.WriteFile(context.Background(), filepath.Join(root, "package.json"), []byte(`{"dependencies":{"lodash":"4.17.20"},"devDependencies":{"vitest":"1.0.0"}}`)); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := fs.WriteFile(context.Background(), filepath.Join(root, "package-lock.json"), []byte(`{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}
	service := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Findings: []shared.Finding{{ID: "OSV-1", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "test", FixVersion: "4.17.21", Fixable: true}}},
		&helpers.StubCache{},
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{audit.NewNPMParser(fs)},
	)
	fixer := NewService(service, &helpers.StubPatchWriter{}, fs, &helpers.StubUI{})
	plan, err := fixer.Plan(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Operations) != 1 {
		t.Fatalf("expected one operation, got %d", len(plan.Operations))
	}
	if plan.Operations[0].ManifestSection != "dependencies" {
		t.Fatalf("expected dependency bump, got section %q", plan.Operations[0].ManifestSection)
	}
	if len(plan.Reasons) != 0 {
		t.Fatalf("expected no explanation buckets for fully plannable scan, got %+v", plan.Reasons)
	}
}

func TestPlanExplainsWhyNoSafeRemediationWasPlanned(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	if err := fs.WriteFile(context.Background(), filepath.Join(root, "package.json"), []byte(`{"dependencies":{"lodash":"4.17.20"}}`)); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := fs.WriteFile(context.Background(), filepath.Join(root, "package-lock.json"), []byte(`{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}
	service := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{Findings: []shared.Finding{{ID: "rule-1", Source: shared.FindingSourceRule, Severity: shared.SeverityHigh, Target: "src/app.tsx", Summary: "manual remediation required"}}},
		&helpers.StubOSV{Findings: []shared.Finding{
			{ID: "OSV-no-fix", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "no fix yet", Fixable: false},
			{ID: "OSV-no-package", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, Target: "unknown-lockfile-entry", Summary: "missing package metadata", FixVersion: "2.0.0", Fixable: true},
		}},
		&helpers.StubCache{},
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{audit.NewNPMParser(fs)},
	)
	fixer := NewService(service, &helpers.StubPatchWriter{}, fs, &helpers.StubUI{})
	plan, err := fixer.Plan(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Operations) != 0 {
		t.Fatalf("expected no operations, got %d", len(plan.Operations))
	}
	if len(plan.Reasons) != 3 {
		t.Fatalf("expected three explanation buckets, got %+v", plan.Reasons)
	}
	if plan.DryRunDiff != "No package.json diff available because no safe manifest changes were planned.\n" {
		t.Fatalf("unexpected diff message: %q", plan.DryRunDiff)
	}
	assertReasonCount(t, plan.Reasons, shared.FixPlanReasonNoFixedVersion, 1)
	assertReasonCount(t, plan.Reasons, shared.FixPlanReasonManualChange, 1)
	assertReasonCount(t, plan.Reasons, shared.FixPlanReasonOutsideScope, 1)
}

func TestDedupeOperationsUsesSemverAwareComparison(t *testing.T) {
	t.Parallel()

	ops := dedupeOperations([]shared.FixOperation{
		{PackageName: "lodash", ProposedVersion: "4.9.0"},
		{PackageName: "lodash", ProposedVersion: "4.18.0"},
	})
	if len(ops) != 1 {
		t.Fatalf("expected one deduplicated operation, got %d", len(ops))
	}
	if ops[0].ProposedVersion != "4.18.0" {
		t.Fatalf("expected highest semver fix, got %q", ops[0].ProposedVersion)
	}
}

func assertReasonCount(t *testing.T, reasons []shared.FixPlanReason, category shared.FixPlanReasonCategory, want int) {
	t.Helper()
	for _, reason := range reasons {
		if reason.Category == category {
			if reason.Count != want {
				t.Fatalf("expected %s count %d, got %d", category, want, reason.Count)
			}
			return
		}
	}
	t.Fatalf("missing reason category %s", category)
}
