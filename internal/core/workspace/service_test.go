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
	mustWriteWorkspaceFile(t, fs, filepath.Join(root, ".gitignore"), "dist/\n")
	mustWriteWorkspaceFile(t, fs, filepath.Join(root, "CLAUDE.md"), "Project instructions\n")

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
	assertContains(t, fs, filepath.Join(root, ".gitignore"), "dist/")
	assertContains(t, fs, filepath.Join(root, ".gitignore"), ".celador/")
	assertContains(t, fs, filepath.Join(root, ".npmrc"), "registry=https://registry.npmjs.org/")
	assertContains(t, fs, filepath.Join(root, ".npmrc"), "ignore-scripts=true")
	assertContains(t, fs, filepath.Join(root, "CLAUDE.md"), "Project instructions")
	assertContains(t, fs, filepath.Join(root, "CLAUDE.md"), "<!-- celador:start -->")
	assertPathMissing(t, fs, filepath.Join(root, "llm.txt"))

	if _, err := service.Run(ctx, root, false, true, false); err != nil {
		t.Fatalf("rerun init: %v", err)
	}
	after, err := fs.ReadFile(ctx, filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE after rerun: %v", err)
	}
	if strings.Count(string(after), "<!-- celador:start -->") != 1 || strings.Count(string(after), "<!-- celador:end -->") != 1 {
		t.Fatalf("expected single managed CLAUDE block after rerun: %s", string(after))
	}
	if !strings.Contains(string(after), "Project instructions") {
		t.Fatalf("CLAUDE rerun lost user-authored prefix: %s", string(after))
	}
}

func TestRunCreatesAgentsWithoutCreatingClaudeOrLLM(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	ctx := context.Background()
	manifest := filepath.Join(root, "package.json")
	lockfile := filepath.Join(root, "package-lock.json")
	configPath := filepath.Join(root, ".celador.yaml")

	mustWriteWorkspaceFile(t, fs, manifest, `{"engines":{"node":"20.0.0"}}`)
	mustWriteWorkspaceFile(t, fs, lockfile, `{"packages":{}}`)
	mustWriteWorkspaceFile(t, fs, configPath, "rules:\n  version: custom\n")

	service := NewService(
		fs,
		helpers.StubDetector{Workspace: shared.Workspace{Root: root, PackageManager: shared.PackageManagerNPM, Lockfiles: []string{lockfile}, ManifestPath: manifest, ConfigPath: configPath}},
		fsadapter.NewIgnoreStore(fs),
		&helpers.StubUI{},
	)

	if _, err := service.Run(ctx, root, false, true, false); err != nil {
		t.Fatalf("run init: %v", err)
	}

	assertContains(t, fs, filepath.Join(root, "AGENTS.md"), "## Celador Supply Chain Security")
	assertPathMissing(t, fs, filepath.Join(root, "CLAUDE.md"))
	assertPathMissing(t, fs, filepath.Join(root, "llm.txt"))
	assertContains(t, fs, filepath.Join(root, ".gitignore"), ".celador/")

	gitignore, err := fs.ReadFile(ctx, filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if strings.Count(string(gitignore), ".celador/") != 1 {
		t.Fatalf("expected .celador/ to be appended once, got %s", string(gitignore))
	}
	if strings.Count(string(gitignore), "coverage/") != 1 {
		t.Fatalf("expected coverage/ to be appended once, got %s", string(gitignore))
	}

	if _, err := service.Run(ctx, root, false, true, false); err != nil {
		t.Fatalf("rerun init: %v", err)
	}

	gitignore, err = fs.ReadFile(ctx, filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore after rerun: %v", err)
	}
	if strings.Count(string(gitignore), ".celador/") != 1 {
		t.Fatalf("expected rerun to avoid duplicate .celador/ entry, got %s", string(gitignore))
	}
}

func TestDetectFallsBackToNPMWhenPackageJSONExistsWithoutLockfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	ctx := context.Background()
	manifest := filepath.Join(root, "package.json")
	mustWriteWorkspaceFile(t, fs, manifest, `{"name":"demo","dependencies":{"react":"18.0.0"}}`)

	detector := NewDetector(fs)
	ws, err := detector.Detect(ctx, root, false, true)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if ws.PackageManager != shared.PackageManagerNPM {
		t.Fatalf("expected npm fallback, got %s", ws.PackageManager)
	}
	if ws.ManifestPath != manifest {
		t.Fatalf("expected manifest path %s, got %s", manifest, ws.ManifestPath)
	}
}

func TestDetectInfersManagerFromWorkspaceFilesWithoutLockfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		files    map[string]string
		want     shared.PackageManager
		manifest string
	}{
		{
			name: "package manager field",
			files: map[string]string{
				"package.json": `{"name":"demo","packageManager":"pnpm@9.0.0"}`,
			},
			want: shared.PackageManagerPNPM,
		},
		{
			name: "pnpm workspace file",
			files: map[string]string{
				"package.json":        `{"name":"demo"}`,
				"pnpm-workspace.yaml": "packages:\n  - apps/*\n",
			},
			want: shared.PackageManagerPNPM,
		},
		{
			name: "bun config file",
			files: map[string]string{
				"package.json": `{"name":"demo"}`,
				"bunfig.toml":  "[install]\nsaveExact=true\n",
			},
			want: shared.PackageManagerBun,
		},
		{
			name: "deno config file",
			files: map[string]string{
				"deno.json": `{"lock":true}`,
			},
			want: shared.PackageManagerDeno,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			fs := fsadapter.NewOSFileSystem(root)
			for name, body := range tc.files {
				mustWriteWorkspaceFile(t, fs, filepath.Join(root, name), body)
			}

			ws, err := NewDetector(fs).Detect(context.Background(), root, false, true)
			if err != nil {
				t.Fatalf("detect: %v", err)
			}
			if ws.PackageManager != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, ws.PackageManager)
			}
		})
	}
}

func TestDetectRemainsUnknownForUnsupportedExplicitManagerWithoutOtherSignals(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	mustWriteWorkspaceFile(t, fs, filepath.Join(root, "package.json"), `{"name":"demo","packageManager":"yarn@4.1.0"}`)

	ws, err := NewDetector(fs).Detect(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if ws.PackageManager != shared.PackageManagerUnknown {
		t.Fatalf("expected unknown package manager, got %s", ws.PackageManager)
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

func assertPathMissing(t *testing.T, fs interface {
	Stat(context.Context, string) (bool, error)
}, path string) {
	t.Helper()
	exists, err := fs.Stat(context.Background(), path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if exists {
		t.Fatalf("expected %s to be absent", path)
	}
}
