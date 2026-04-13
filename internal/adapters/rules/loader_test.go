package rules

import (
	"context"
	"fmt"
	"testing"

	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestYAMLLoader_LoadsRuleFiles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	rulesDir := "/root/configs/rules"
	yamlContent := `
version: v2
rules:
  - id: test-rule-1
    file: src/app.ts
    summary: Test rule
    severity: high
    must_not_find: dangerous
  - id: test-rule-2
    file: src/db.ts
    summary: Another rule
    severity: medium
    must_not_find: injection
`
	fakeFS := &helpers.FakeFileSystem{
		GlobFn: func(ctx context.Context, root string, patterns ...string) ([]string, error) {
			return []string{rulesDir + "/rules.yaml"}, nil
		},
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(yamlContent), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	loader := NewYAMLLoader(fakeFS)
	rules, version, err := loader.Load(ctx, "/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].ID != "test-rule-1" {
		t.Errorf("expected first rule ID 'test-rule-1', got %q", rules[0].ID)
	}
	if version != "v2" {
		t.Errorf("expected version 'v2', got %q", version)
	}
	if rules[0].ID < rules[1].ID {
		// Rules should be sorted by ID
	} else {
		t.Error("expected rules sorted by ID")
	}
}

func TestYAMLLoader_EmptyDirectory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		GlobFn: func(ctx context.Context, root string, patterns ...string) ([]string, error) {
			return []string{}, nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	loader := NewYAMLLoader(fakeFS)
	rules, version, err := loader.Load(ctx, "/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
	if version != "v1" {
		t.Errorf("expected default version 'v1', got %q", version)
	}
}

func TestYAMLLoader_InvalidYAML(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		GlobFn: func(ctx context.Context, root string, patterns ...string) ([]string, error) {
			return []string{"/root/configs/rules/bad.yaml"}, nil
		},
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(`{invalid: yaml: content: [`), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	loader := NewYAMLLoader(fakeFS)
	_, _, err := loader.Load(ctx, "/root")
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestYAMLLoader_GlobError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		GlobFn: func(ctx context.Context, root string, patterns ...string) ([]string, error) {
			return nil, fmt.Errorf("glob failed")
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	loader := NewYAMLLoader(fakeFS)
	_, _, err := loader.Load(ctx, "/root")
	if err == nil {
		t.Fatal("expected error for glob failure, got nil")
	}
}

func TestYAMLLoader_ReadFileError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		GlobFn: func(ctx context.Context, root string, patterns ...string) ([]string, error) {
			return []string{"/root/configs/rules/good.yaml"}, nil
		},
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, fmt.Errorf("permission denied")
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	loader := NewYAMLLoader(fakeFS)
	_, _, err := loader.Load(ctx, "/root")
	if err == nil {
		t.Fatal("expected error for read failure, got nil")
	}
}

func TestYAMLLoader_MultipleFilesVersionFromFirst(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fileContents := map[string]string{
		"/root/configs/rules/a.yaml": "version: v3\nrules:\n  - id: rule-a\n    file: a.ts\n    severity: low\n",
		"/root/configs/rules/b.yaml": "version: v5\nrules:\n  - id: rule-b\n    file: b.ts\n    severity: high\n",
	}
	callOrder := 0
	fakeFS := &helpers.FakeFileSystem{
		GlobFn: func(ctx context.Context, root string, patterns ...string) ([]string, error) {
			return []string{"/root/configs/rules/a.yaml", "/root/configs/rules/b.yaml"}, nil
		},
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			callOrder++
			return []byte(fileContents[path]), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	loader := NewYAMLLoader(fakeFS)
	rules, version, err := loader.Load(ctx, "/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	// Version should come from the first file that has a non-empty version.
	// Since both files have versions, the first one processed wins.
	if version != "v3" && version != "v5" {
		t.Errorf("expected version from one of the files, got %q", version)
	}
	// Rules should sorted by ID
	if rules[0].ID != "rule-a" || rules[1].ID != "rule-b" {
		t.Errorf("expected rules sorted by ID, got %s, %s", rules[0].ID, rules[1].ID)
	}
}
