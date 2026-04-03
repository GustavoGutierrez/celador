package e2e

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/app"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestInitScanFixInstallCommands(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "package.json"), `{"engines":{"node":"20.0.0"},"dependencies":{"lodash":"4.17.20"},"devDependencies":{"vite":"5.0.0"}}`)
	mustWriteFile(t, filepath.Join(root, "package-lock.json"), `{"packages":{"node_modules/lodash":{"version":"4.17.20"}}}`)
	mustWriteFile(t, filepath.Join(root, "vite.config.ts"), `export default { build: { sourcemap: true } }`)
	mustWriteFile(t, filepath.Join(root, "configs", "rules", "vite.yaml"), "version: v1\nrules:\n  - id: vite-production-sourcemap\n    frameworks: [vite]\n    file: vite.config.ts\n    mustNotFind: \"sourcemap: true\"\n    severity: high\n    summary: \"No sourcemaps\"\n")

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(previousWD) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	bootstrap, err := app.NewBootstrap(context.Background(), []string{"init"})
	if err != nil {
		t.Fatalf("bootstrap init: %v", err)
	}
	buffer := &bytes.Buffer{}
	bootstrap.OverrideOutput(buffer)
	if err := bootstrap.Execute(context.Background()); err != nil {
		t.Fatalf("init execute: %v", err)
	}
	for _, args := range [][]string{{"scan"}, {"fix", "--diff"}} {
		bootstrap, err := app.NewBootstrap(context.Background(), args)
		if err != nil {
			t.Fatalf("bootstrap %v: %v", args, err)
		}
		bootstrap.OverrideOutput(buffer)
		_ = bootstrap.Execute(context.Background())
	}
	bootstrap, err = app.NewBootstrap(context.Background(), []string{"install", "lodash", "--yes"})
	if err != nil {
		t.Fatalf("bootstrap install: %v", err)
	}
	bootstrap.OverrideOutput(buffer)
	bootstrap.OverridePackageMetadata(helpers.StubMetadata{Assessment: shared.InstallAssessment{Package: "lodash", Risk: shared.SeverityLow}})
	bootstrap.OverridePackageManager(&helpers.StubPM{})
	_ = bootstrap.Execute(context.Background())
}

func mustWriteFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
