package fs

import (
	"context"
	"strings"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestPatchWriter_Apply_BumpDependency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var written []byte

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(`{"dependencies":{"lodash":"4.17.20"}}`), nil
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			written = data
			return nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	writer := NewPatchWriter(fakeFS)
	workspace := shared.Workspace{ManifestPath: "/root/package.json"}
	plan := shared.FixPlan{
		Operations: []shared.FixOperation{{
			PackageName:    "lodash",
			CurrentVersion: "4.17.20",
			ProposedVersion: "4.17.21",
			Strategy:       "bump",
			ManifestSection: "dependencies",
		}},
	}

	err := writer.Apply(ctx, workspace, plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(written), `"lodash": "4.17.21"`) {
		t.Errorf("expected lodash bumped to 4.17.21, got: %s", string(written))
	}
}

func TestPatchWriter_Apply_BumpDevDependency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var written []byte

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(`{"devDependencies":{"typescript":"4.9.0"}}`), nil
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			written = data
			return nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	writer := NewPatchWriter(fakeFS)
	workspace := shared.Workspace{ManifestPath: "/root/package.json"}
	plan := shared.FixPlan{
		Operations: []shared.FixOperation{{
			PackageName:    "typescript",
			CurrentVersion: "4.9.0",
			ProposedVersion: "5.0.0",
			Strategy:       "bump",
			ManifestSection: "devDependencies",
		}},
	}

	err := writer.Apply(ctx, workspace, plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(written), `"typescript": "5.0.0"`) {
		t.Errorf("expected typescript bumped to 5.0.0, got: %s", string(written))
	}
}

func TestPatchWriter_Apply_AddOverride(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var written []byte

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(`{"dependencies":{}}`), nil
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			written = data
			return nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	writer := NewPatchWriter(fakeFS)
	workspace := shared.Workspace{ManifestPath: "/root/package.json"}
	plan := shared.FixPlan{
		Operations: []shared.FixOperation{{
			PackageName:    "risky-pkg",
			CurrentVersion: "1.0.0",
			ProposedVersion: "2.0.0",
			Strategy:       "override",
		}},
	}

	err := writer.Apply(ctx, workspace, plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(written), `"overrides"`) {
		t.Errorf("expected overrides section, got: %s", string(written))
	}
	if !strings.Contains(string(written), `"risky-pkg": "2.0.0"`) {
		t.Errorf("expected risky-pkg in overrides, got: %s", string(written))
	}
}

func TestPatchWriter_Apply_MissingManifestPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	writer := NewPatchWriter(fakeFS)
	workspace := shared.Workspace{ManifestPath: ""}
	plan := shared.FixPlan{Operations: []shared.FixOperation{}}

	err := writer.Apply(ctx, workspace, plan)
	if err == nil {
		t.Fatal("expected error for missing manifest path, got nil")
	}
	if !strings.Contains(err.Error(), "package.json") {
		t.Errorf("expected error to mention package.json, got: %v", err)
	}
}

func TestPatchWriter_Apply_InvalidJSON(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(`{invalid json}`), nil
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			return nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	writer := NewPatchWriter(fakeFS)
	workspace := shared.Workspace{ManifestPath: "/root/package.json"}
	plan := shared.FixPlan{Operations: []shared.FixOperation{}}

	err := writer.Apply(ctx, workspace, plan)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestPatchWriter_Preview_ReturnsDiff(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	writer := NewPatchWriter(fakeFS)
	workspace := shared.Workspace{ManifestPath: "/root/package.json"}
	expectedDiff := "--- package.json\n+++ package.json\n-lodash@4.17.20\n+lodash@4.17.21"
	plan := shared.FixPlan{DryRunDiff: expectedDiff}

	result, err := writer.Preview(ctx, workspace, plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedDiff {
		t.Errorf("expected diff %q, got %q", expectedDiff, result)
	}
}

func TestRenderPlanDiff_EmptyPlan(t *testing.T) {
	t.Parallel()

	result := RenderPlanDiff(nil)
	if !strings.Contains(result, "No package.json diff") {
		t.Errorf("expected friendly message for empty plan, got: %s", result)
	}
}

func TestRenderPlanDiff_WithOperations(t *testing.T) {
	t.Parallel()

	ops := []shared.FixOperation{
		{PackageName: "lodash", CurrentVersion: "4.17.20", ProposedVersion: "4.17.21", Strategy: "bump"},
		{PackageName: "express", CurrentVersion: "4.18.0", ProposedVersion: "4.18.2", Strategy: "bump"},
	}

	result := RenderPlanDiff(ops)
	if !strings.Contains(result, "lodash") {
		t.Error("expected lodash in diff")
	}
	if !strings.Contains(result, "express") {
		t.Error("expected express in diff")
	}
	if !strings.Contains(result, "---") {
		t.Error("expected unified diff format")
	}
}
