package workspace

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestRunPreservesUnrelatedWorkspaceContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	ctx := context.Background()
	manifest := filepath.Join(root, "package.json")
	lockfile := filepath.Join(root, "package-lock.json")
	configPath := filepath.Join(root, ".celador.yaml")

	mustWriteWorkspaceFile(t, fs, manifest, `{"engines":{"node":"20.0.0"}}`)
	mustWriteWorkspaceFile(t, fs, lockfile, `{"packages":{}}`)
	mustWriteWorkspaceFile(t, fs, configPath, "custom:\n  keep: true\ncache:\n  ttl: 1h\n")
	mustWriteWorkspaceFile(t, fs, filepath.Join(root, ".npmrc"), "registry=https://registry.npmjs.org/\nignore-scripts=false\n")
	mustWriteWorkspaceFile(t, fs, filepath.Join(root, "llm.txt"), "Project instructions\n")

	service := NewService(
		fs,
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{lockfile}, ManifestPath: manifest, ConfigPath: configPath}},
		fsadapter.NewIgnoreStore(fs),
		&helpers.StubUI{},
	)

	if _, err := service.Run(ctx, root, false, true, false); err != nil {
		t.Fatalf("run init: %v", err)
	}

	assertContains(t, fs, configPath, "custom:")
	assertContains(t, fs, configPath, "keep: true")
	assertContains(t, fs, configPath, "ttl: 24h")
	assertContains(t, fs, filepath.Join(root, ".npmrc"), "registry=https://registry.npmjs.org/")
	assertContains(t, fs, filepath.Join(root, ".npmrc"), "ignore-scripts=true")
	assertContains(t, fs, filepath.Join(root, "llm.txt"), "Project instructions")
	assertContains(t, fs, filepath.Join(root, "llm.txt"), "# celador:start")

	if _, err := service.Run(ctx, root, false, true, false); err != nil {
		t.Fatalf("rerun init: %v", err)
	}
	after, err := fs.ReadFile(ctx, filepath.Join(root, "llm.txt"))
	if err != nil {
		t.Fatalf("read llm after rerun: %v", err)
	}
	if strings.Count(string(after), "# celador:start") != 1 || strings.Count(string(after), "# celador:end") != 1 {
		t.Fatalf("expected single managed llm block after rerun: %s", string(after))
	}
	if !strings.Contains(string(after), "Project instructions") {
		t.Fatalf("llm rerun lost user-authored prefix: %s", string(after))
	}
}

func mustWriteWorkspaceFile(t *testing.T, fs interface {
	WriteFile(context.Context, string, []byte) error
}, path string, body string) {
	t.Helper()
	if err := fs.WriteFile(context.Background(), path, []byte(body)); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertContains(t *testing.T, fs interface {
	ReadFile(context.Context, string) ([]byte, error)
}, path string, want string) {
	t.Helper()
	body, err := fs.ReadFile(context.Background(), path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(body), want) {
		t.Fatalf("expected %s to contain %q, got %s", path, want, string(body))
	}
}
