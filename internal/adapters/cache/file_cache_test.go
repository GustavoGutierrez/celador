package cache

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestGetScanIgnoresLegacySchemaEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	cache := NewFileCache(fs, root, helpers.StubClock{Value: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)})
	path := filepath.Join(root, "scan-legacy.json")

	if err := fs.WriteFile(context.Background(), path, []byte(`{"findings":[{"id":"legacy-osv"}]}`)); err != nil {
		t.Fatalf("write legacy scan cache: %v", err)
	}

	result, ok, err := cache.GetScan(context.Background(), "legacy")
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	if ok {
		t.Fatalf("expected legacy scan cache miss, got hit: %+v", result)
	}
}

func TestGetOSVIgnoresLegacySchemaEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	cache := NewFileCache(fs, root, helpers.StubClock{Value: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)})
	path := filepath.Join(root, "osv-legacy.json")

	if err := fs.WriteFile(context.Background(), path, []byte(`{"findings":[{"id":"legacy-osv"}],"expiresAt":"2026-04-05T00:00:00Z"}`)); err != nil {
		t.Fatalf("write legacy osv cache: %v", err)
	}

	findings, ok, expiresAt, err := cache.GetOSV(context.Background(), "legacy")
	if err != nil {
		t.Fatalf("get osv: %v", err)
	}
	if ok {
		t.Fatalf("expected legacy osv cache miss, got hit: findings=%+v expiresAt=%s", findings, expiresAt)
	}
}

func TestPutScanPersistsSchemaVersion(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	cache := NewFileCache(fs, root, helpers.StubClock{Value: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)})
	result := shared.ScanResult{Fingerprint: "fp-123", Findings: []shared.Finding{{ID: "OSV-1"}}}

	if err := cache.PutScan(context.Background(), "fresh", result); err != nil {
		t.Fatalf("put scan: %v", err)
	}

	got, ok, err := cache.GetScan(context.Background(), "fresh")
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	if !ok {
		t.Fatalf("expected versioned scan cache hit")
	}
	if got.Fingerprint != result.Fingerprint || len(got.Findings) != 1 || got.Findings[0].ID != "OSV-1" {
		t.Fatalf("unexpected scan cache payload: %+v", got)
	}
}
