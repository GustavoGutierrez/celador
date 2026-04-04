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

func TestPlanUsesRenderedFindingRepresentatives(t *testing.T) {
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
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Findings: []shared.Finding{
			{ID: "GHSA-dup", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "node_modules/lodash", Summary: "duplicate advisory", FixVersion: "4.17.21", Fixable: true},
			{ID: "GHSA-dup", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "node_modules/lodash", Summary: "duplicate advisory", FixVersion: "4.17.21", Fixable: true},
			{ID: "GHSA-no-fix", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "node_modules/lodash", Summary: "upstream fix pending", Fixable: false},
			{ID: "GHSA-no-fix", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "node_modules/lodash", Summary: "upstream fix pending", Fixable: false},
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
	if len(plan.Operations) != 1 {
		t.Fatalf("expected one operation after rendered dedupe, got %d", len(plan.Operations))
	}
	if plan.Operations[0].ProposedVersion != "4.17.21" {
		t.Fatalf("expected highest rendered fix version, got %q", plan.Operations[0].ProposedVersion)
	}
	assertReasonCount(t, plan.Reasons, shared.FixPlanReasonNoFixedVersion, 1)
}

func TestPlanLeavesNonConservativeUpgradesForReview(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	if err := fs.WriteFile(context.Background(), filepath.Join(root, "package.json"), []byte(`{"dependencies":{"lodash":"4.17.20","next":"15.2.4","jspdf":"^3.0.3","nodemailer":"^7.0.6"}}`)); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := fs.WriteFile(context.Background(), filepath.Join(root, "package-lock.json"), []byte(`{"packages":{"node_modules/lodash":{"version":"4.17.20"},"node_modules/next":{"version":"15.2.4"},"node_modules/jspdf":{"version":"3.0.3"},"node_modules/nodemailer":{"version":"7.0.6"},"node_modules/minimist":{"version":"1.2.7"}}}`)); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	service := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Findings: []shared.Finding{
			{ID: "OSV-safe", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "safe patch", FixVersion: "4.17.21", Fixable: true},
			{ID: "OSV-prerelease", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "next", Target: "next", Summary: "prerelease only", FixVersion: "15.6.0-canary.61", Fixable: true},
			{ID: "OSV-major-jspdf", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "jspdf", Target: "jspdf", Summary: "major bump", FixVersion: "4.2.1", Fixable: true},
			{ID: "OSV-major-nodemailer", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "nodemailer", Target: "nodemailer", Summary: "major bump", FixVersion: "8.0.4", Fixable: true},
			{ID: "OSV-override", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "minimist", Target: "minimist", Summary: "safe override", FixVersion: "1.2.8", Fixable: true},
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
	if len(plan.Operations) != 2 {
		t.Fatalf("expected two safe operations, got %d", len(plan.Operations))
	}
	if plan.Operations[0].PackageName != "lodash" || plan.Operations[1].PackageName != "minimist" {
		t.Fatalf("expected lodash bump and minimist override, got %+v", plan.Operations)
	}
	if plan.Operations[1].Strategy != "override" {
		t.Fatalf("expected minimist to stay an override, got %+v", plan.Operations[1])
	}
	assertReasonCount(t, plan.Reasons, shared.FixPlanReasonNonConservativeUpgrade, 3)
	assertReasonMissing(t, plan.Reasons, shared.FixPlanReasonManualChange)
	assertReasonMissing(t, plan.Reasons, shared.FixPlanReasonOutsideScope)
	assertReasonMissing(t, plan.Reasons, shared.FixPlanReasonNoFixedVersion)
}

func TestPlanRejectsPrereleaseOverrideWithoutManifestVersion(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	if err := fs.WriteFile(context.Background(), filepath.Join(root, "package.json"), []byte(`{"dependencies":{"lodash":"4.17.20"}}`)); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := fs.WriteFile(context.Background(), filepath.Join(root, "package-lock.json"), []byte(`{"packages":{"node_modules/lodash":{"version":"4.17.20"},"node_modules/minimist":{"version":"1.2.7"}}}`)); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	service := audit.NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		helpers.StubIgnore{},
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Findings: []shared.Finding{{ID: "OSV-prerelease-override", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "minimist", Target: "minimist", Summary: "transitive prerelease", FixVersion: "1.2.9-rc.1", Fixable: true}}},
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
		t.Fatalf("expected no auto-planned operations, got %+v", plan.Operations)
	}
	assertReasonCount(t, plan.Reasons, shared.FixPlanReasonNonConservativeUpgrade, 1)
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

func assertReasonMissing(t *testing.T, reasons []shared.FixPlanReason, category shared.FixPlanReasonCategory) {
	t.Helper()
	for _, reason := range reasons {
		if reason.Category == category {
			t.Fatalf("expected reason category %s to be absent, got %+v", category, reason)
		}
	}
}
