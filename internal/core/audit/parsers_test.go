package audit

import (
	"context"
	"path/filepath"
	"testing"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
	"github.com/GustavoGutierrez/celador/test/helpers"
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

func TestParserSupports(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		parser   interface{ Supports(string) bool }
		path     string
		expected bool
	}{
		{"npm_match", NewNPMParser(nil), "package-lock.json", true},
		{"npm_no_match", NewNPMParser(nil), "pnpm-lock.yaml", false},
		{"pnpm_match", NewPNPMParser(nil), "pnpm-lock.yaml", true},
		{"pnpm_no_match", NewPNPMParser(nil), "package-lock.json", false},
		{"bun_match", NewBunParser(nil), "bun.lock", true},
		{"bun_lockb_no_match", NewBunParser(nil), "bun.lockb", false},
		{"bun_no_match", NewBunParser(nil), "package.json", false},
		{"deno_match", NewDenoParser(nil), "deno.lock", true},
		{"deno_no_match", NewDenoParser(nil), "package-lock.json", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.parser.Supports(tt.path)
			if result != tt.expected {
				t.Errorf("Supports(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestParserFileSystem(t *testing.T) {
	t.Parallel()
	fakeFS := &helpers.FakeFileSystem{}
	tests := []struct {
		name   string
		parser interface{ FileSystem() ports.FileSystem }
	}{
		{"npm", NewNPMParser(fakeFS)},
		{"pnpm", NewPNPMParser(fakeFS)},
		{"bun", NewBunParser(fakeFS)},
		{"deno", NewDenoParser(fakeFS)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.parser.FileSystem()
			if fs != fakeFS {
				t.Errorf("expected same filesystem, got different")
			}
		})
	}
}

func TestParser_ParseReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, context.Canceled
		},
	}
	parser := NewNPMParser(fakeFS)
	_, err := parser.Parse(ctx, shared.Workspace{}, "/root/package-lock.json")
	if err == nil {
		t.Fatal("expected error for read failure, got nil")
	}
}

func TestParser_ParseInvalidJSON(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(`{invalid`), nil
		},
	}
	parser := NewNPMParser(fakeFS)
	_, err := parser.Parse(ctx, shared.Workspace{}, "/root/package-lock.json")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParser_ParseInvalidYAML(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(`{invalid: yaml: [`), nil
		},
	}
	parser := NewPNPMParser(fakeFS)
	_, err := parser.Parse(ctx, shared.Workspace{}, "/root/pnpm-lock.yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
