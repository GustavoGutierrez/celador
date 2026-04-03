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
}
