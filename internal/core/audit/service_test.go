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

func TestCacheKeyForDependencies_Error(t *testing.T) {
	t.Parallel()
	// Create a dependency with an unmarshalable field to trigger json.Marshal error
	// (In practice this won't happen, but we test the error path)
	// Since shared.Dependency is simple, we just verify the happy path works:
	deps := []shared.Dependency{{Name: "test", Version: "1.0.0", Ecosystem: "npm"}}
	key, err := cacheKeyForDependencies(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == "" {
		t.Fatal("expected non-empty cache key")
	}
}

func TestParseDependencies_NoParserForFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fakeFS := &helpers.FakeFileSystem{}
	parser := NewNPMParser(fakeFS)

	service := NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: "/root"}},
		nil, nil, nil, nil, nil,
		helpers.StubClock{Value: time.Now()},
		time.Hour,
		[]ports.LockfileParser{parser},
	)

	ws := shared.Workspace{Root: "/root", Lockfiles: []string{"/root/unknown.lock"}}
	_, err := service.parseDependencies(ctx, ws)
	if err == nil {
		t.Fatal("expected error for unsupported lockfile, got nil")
	}
}

func TestParseDependencies_ParseError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(`{invalid`), nil
		},
	}
	parser := NewNPMParser(fakeFS)

	service := NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: "/root"}},
		nil, nil, nil, nil, nil,
		helpers.StubClock{Value: time.Now()},
		time.Hour,
		[]ports.LockfileParser{parser},
	)

	ws := shared.Workspace{Root: "/root", Lockfiles: []string{"/root/package-lock.json"}}
	_, err := service.parseDependencies(ctx, ws)
	if err == nil {
		t.Fatal("expected error for parse failure, got nil")
	}
}

func TestSortFindings(t *testing.T) {
	t.Parallel()
	findings := []shared.Finding{
		{ID: "GHSA-zzzz", PackageName: "b-pkg"},
		{ID: "GHSA-aaaa", PackageName: "a-pkg"},
		{ID: "GHSA-aaaa", PackageName: "c-pkg"},
	}
	sortFindings(findings)
	// Should be sorted by ID, then by PackageName
	if findings[0].ID != "GHSA-aaaa" || findings[0].PackageName != "a-pkg" {
		t.Errorf("expected first finding GHSA-aaaa/a-pkg, got %s/%s", findings[0].ID, findings[0].PackageName)
	}
	if findings[1].PackageName != "c-pkg" {
		t.Errorf("expected second finding c-pkg, got %s", findings[1].PackageName)
	}
	if findings[2].ID != "GHSA-zzzz" {
		t.Errorf("expected last finding GHSA-zzzz, got %s", findings[2].ID)
	}
}

func TestParserFS(t *testing.T) {
	t.Parallel()
	fakeFS := &helpers.FakeFileSystem{}
	parsers := []ports.LockfileParser{
		NewNPMParser(fakeFS),
		NewPNPMParser(fakeFS),
	}
	fs := parserFS(parsers)
	if fs != fakeFS {
		t.Error("expected parserFS to return the filesystem from the first parser")
	}
	if parserFS(nil) != nil {
		t.Error("expected parserFS to return nil for empty slice")
	}
}

func TestLoadOSVFindings_CacheExpired(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	clockTime := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	expiredTime := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC) // Before clock

	cache := &helpers.StubCache{
		OSV:        map[string][]shared.Finding{"key": {{ID: "OSV-expired"}}},
		OSVExpiry:  map[string]time.Time{"key": expiredTime},
	}

	service := &Service{
		cache: cache,
		clock: helpers.StubClock{Value: clockTime},
	}

	findings, needsRefresh, fromCache, err := service.loadOSVFindings(ctx, "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if !needsRefresh {
		t.Error("expected needsRefresh=true for expired cache")
	}
	if fromCache {
		t.Error("expected fromCache=false for expired cache")
	}
}

func TestLoadOSVFindings_CacheValid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	clockTime := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	futureTime := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC) // After clock

	cache := &helpers.StubCache{
		OSV:        map[string][]shared.Finding{"key": {{ID: "OSV-valid"}}},
		OSVExpiry:  map[string]time.Time{"key": futureTime},
	}

	service := &Service{
		cache: cache,
		clock: helpers.StubClock{Value: clockTime},
	}

	findings, needsRefresh, fromCache, err := service.loadOSVFindings(ctx, "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if needsRefresh {
		t.Error("expected needsRefresh=false for valid cache")
	}
	if !fromCache {
		t.Error("expected fromCache=true for valid cache")
	}
}

func TestLoadOSVFindings_CacheMiss(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cache := &helpers.StubCache{OSV: map[string][]shared.Finding{}}

	service := &Service{
		cache: cache,
		clock: helpers.StubClock{Value: time.Now()},
	}

	findings, needsRefresh, fromCache, err := service.loadOSVFindings(ctx, "missing-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if findings != nil {
		t.Errorf("expected nil findings for cache miss, got %v", findings)
	}
	if needsRefresh || fromCache {
		t.Error("expected both flags false for cache miss")
	}
}

func TestLoadOSVFindings_CacheError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cache := &helpers.StubCache{OSVErr: context.DeadlineExceeded}

	service := &Service{
		cache: cache,
		clock: helpers.StubClock{Value: time.Now()},
	}

	_, _, _, err := service.loadOSVFindings(ctx, "key")
	if err == nil {
		t.Fatal("expected error from cache, got nil")
	}
}

func TestServiceRun_WithOSVQueryAndRuleEvaluation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWrite(t, fs, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"}}`)
	mustWrite(t, fs, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)
	mustWrite(t, fs, filepath.Join(root, ".celadorignore"), "")
	mustWrite(t, fs, filepath.Join(root, "configs", "rules", "test.yaml"), "version: v1\nrules: []\n")

	cache := &helpers.StubCache{}
	osv := &helpers.StubOSV{Findings: []shared.Finding{
		{ID: "GHSA-test", Source: shared.FindingSourceOSV, Severity: shared.SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "Test vuln"},
	}}
	service := NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{filepath.Join(root, "package-lock.json")}, ManifestPath: filepath.Join(root, "package.json")}},
		fsadapter.NewIgnoreStore(fs),
		helpers.StubRuleLoader{Version: "v1"},
		helpers.StubRuleEvaluator{},
		osv,
		cache,
		helpers.StubClock{Value: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
		24*time.Hour,
		[]ports.LockfileParser{NewNPMParser(fs)},
	)

	result, err := service.Run(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].ID != "GHSA-test" {
		t.Errorf("expected finding ID 'GHSA-test', got %q", result.Findings[0].ID)
	}
	if osv.Calls != 1 {
		t.Errorf("expected 1 OSV query call, got %d", osv.Calls)
	}
}

func TestReadLockfileBody_ReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, context.Canceled
		},
	}
	_, err := readLockfileBody(ctx, fakeFS, []string{"/root/package-lock.json"})
	if err == nil {
		t.Fatal("expected error for read failure, got nil")
	}
}
