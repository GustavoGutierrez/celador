package audit

import (
	"context"
	"path/filepath"
	"testing"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestParsers(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "test", "fixtures", "lockfiles")
	fs := fsadapter.NewOSFileSystem(root)
	ws := shared.Workspace{ManifestPath: "package.json"}
	tests := []struct {
		name   string
		parser interface {
			Parse(context.Context, shared.Workspace, string) ([]shared.Dependency, error)
		}
		path string
	}{
		{name: "npm", parser: NewNPMParser(fs), path: filepath.Join(root, "package-lock.json")},
		{name: "pnpm", parser: NewPNPMParser(fs), path: filepath.Join(root, "pnpm-lock.yaml")},
		{name: "bun", parser: NewBunParser(fs), path: filepath.Join(root, "bun.lock")},
		{name: "deno", parser: NewDenoParser(fs), path: filepath.Join(root, "deno.lock")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, err := tt.parser.Parse(context.Background(), ws, tt.path)
			if err != nil {
				t.Fatalf("parse %s: %v", tt.name, err)
			}
			if len(deps) == 0 {
				t.Fatalf("expected dependencies for %s", tt.name)
			}
		})
	}
}
