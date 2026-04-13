package fs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestWriteManagedSection_PropagatesReadErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, fmt.Errorf("permission denied")
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, os.ErrNotExist
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			return nil
		},
	}

	err := WriteManagedSection(ctx, fakeFS, "/root/test.md", "<!-- start -->", "<!-- end -->", "content")
	if err == nil {
		t.Fatal("expected error when file cannot be read, got nil")
	}

	expectedMsg := "permission denied"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error to contain %q, got: %v", expectedMsg, err)
	}
}

func TestWriteManagedSection_CreatesWhenNotExists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var writtenData []byte

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, os.ErrNotExist
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, os.ErrNotExist
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			writtenData = data
			return nil
		},
	}

	block := "<!-- celador:start -->\nmanaged content\n<!-- celador:end -->\n"
	err := WriteManagedSection(ctx, fakeFS, "/root/test.md", "<!-- celador:start -->", "<!-- celador:end -->", block)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writtenData) == 0 {
		t.Fatal("expected file to be written, got empty data")
	}

	if !strings.Contains(string(writtenData), "managed content") {
		t.Errorf("expected written content to contain 'managed content', got: %s", string(writtenData))
	}
}

func TestWriteManagedSection_ReplacesExistingBlock(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var writtenData []byte

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte("prefix\n<!-- celador:start -->\nold content\n<!-- celador:end -->\nsuffix"), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			writtenData = data
			return nil
		},
	}

	block := "<!-- celador:start -->\nnew content\n<!-- celador:end -->"
	err := WriteManagedSection(ctx, fakeFS, "/root/test.md", "<!-- celador:start -->", "<!-- celador:end -->", block)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(writtenData)
	if !strings.Contains(result, "new content") {
		t.Errorf("expected written content to contain 'new content', got: %s", result)
	}
	if strings.Contains(result, "old content") {
		t.Errorf("expected written content to NOT contain 'old content', got: %s", result)
	}
	if !strings.Contains(result, "prefix") || !strings.Contains(result, "suffix") {
		t.Errorf("expected written content to preserve prefix/suffix, got: %s", result)
	}
}

func TestValidateWorkspacePath_AcceptsValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		path  string
		root  string
	}{
		{"direct child", "/root/file.txt", "/root"},
		{"nested child", "/root/subdir/file.txt", "/root"},
		{"same path", "/root", "/root"},
		{"with trailing separator", "/root/", "/root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkspacePath(tt.path, tt.root)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateWorkspacePath_RejectsEscapeAttempts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		root string
	}{
		{"parent dir", "/root/../etc/passwd", "/root"},
		{"sibling dir", "/root/../sibling/file.txt", "/root"},
		{"absolute escape", "/etc/passwd", "/root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkspacePath(tt.path, tt.root)
			if err == nil {
				t.Error("expected error for path escape, got nil")
			}
		})
	}
}
