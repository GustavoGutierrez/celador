package audit

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestServiceRunHonorsIgnoreAndCaches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWrite(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"}}`)
	mustWrite(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)
	mustWrite(t, fs, filepath.Join(root, ".celadorignore"), "OSV-1|accepted|2099-01-01\n")
	mustWrite(t, fs, filepath.Join(root, "configs", "rules", "test.yaml"), "version: v1\nrules: []\n")

	parsers := []ports.LockfileParser{NewNPMParser(fs)}
	service := NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		fsadapter.NewIgnoreStore(fs),
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Findings: []shared.Finding{{ID: "OSV-1", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "test"}}},
		&helpers.StubCache{},
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		parsers,
	)
	result, err := service.Run(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if len(result.Findings) != 0 || result.IgnoredCount != 1 {
		t.Fatalf("unexpected findings result: %+v", result)
	}
}

func TestServiceRunUsesOSVTTLCache(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWrite(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"}}`)
	mustWrite(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)
	mustWrite(t, fs, filepath.Join(root, "configs", "rules", "test.yaml"), "version: v1\nrules: []\n")

	parser := NewNPMParser(fs)
	deps, err := parser.Parse(context.Background(), shared.Workspace{Root: root, ManifestPath: filepath.Join(root, "package.json")}, filepath.Join(root, "package-lock.json"))
	if err != nil {
		t.Fatalf("parse dependencies: %v", err)
	}
	osvKey, err := cacheKeyForDependencies(deps)
	if err != nil {
		t.Fatalf("cache key: %v", err)
	}
	cache := &helpers.StubCache{
		OSV: map[string][]shared.Finding{
			osvKey: {{ID: "OSV-CACHED", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "cached"}},
		},
		OSVExpiry: map[string]time.Time{osvKey: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)},
	}
	osv := &helpers.StubOSV{Findings: []shared.Finding{{ID: "OSV-LIVE", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "live"}}}
	service := NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		fsadapter.NewIgnoreStore(fs),
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		osv,
		cache,
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{parser},
	)

	result, err := service.Run(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if osv.Calls != 0 {
		t.Fatalf("expected OSV query to be skipped when ttl cache is warm")
	}
	if !result.FromCache || result.OfflineFallback {
		t.Fatalf("expected cached online result, got %+v", result)
	}
	if len(result.Findings) != 1 || result.Findings[0].ID != "OSV-CACHED" {
		t.Fatalf("unexpected findings: %+v", result.Findings)
	}
}

func TestServiceRunFallsBackOfflineToWarmScanCache(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWrite(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"}}`)
	mustWrite(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)
	mustWrite(t, fs, filepath.Join(root, ".celadorignore"), "")
	mustWrite(t, fs, filepath.Join(root, "configs", "rules", "test.yaml"), "version: v1\nrules: []\n")

	lockText, err := readLockfileBody(context.Background(), fs, []string{filepath.Join(root, "package-lock.json")})
	if err != nil {
		t.Fatalf("read lock text: %v", err)
	}
	fingerprint := Fingerprint(lockText, "[]", "v1")
	cache := &helpers.StubCache{Scan: map[string]shared.ScanResult{fingerprint: {Fingerprint: fingerprint, Findings: []shared.Finding{{ID: "OSV-SCAN", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "scan cache"}}}}}
	service := NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		fsadapter.NewIgnoreStore(fs),
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		&helpers.StubOSV{Err: context.DeadlineExceeded},
		cache,
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{NewNPMParser(fs)},
	)

	result, err := service.Run(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if !result.FromCache || !result.OfflineFallback {
		t.Fatalf("expected offline fallback result, got %+v", result)
	}
}

func mustWrite(t *testing.T, fs interface {
	WriteFile(context.Context, string, []byte) error
}, path string, body string) {
	t.Helper()
	if err := fs.WriteFile(context.Background(), path, []byte(body)); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
