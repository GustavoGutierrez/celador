package workspace

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestEnsureIgnoreFiles_PropagatesReadErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create a filesystem where reading .gitignore will fail
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			if path == "/root/.gitignore" {
				return nil, fmt.Errorf("permission denied")
			}
			if path == "/root/.npmignore" {
				return nil, os.ErrNotExist
			}
			return []byte{}, nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, os.ErrNotExist
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			return nil
		},
	}

	svc := NewService(fakeFS, nil, nil, nil, nil)
	ws := shared.Workspace{
		Root:           "/root",
		ManifestPath:   "/root/package.json",
		PackageManager: shared.PackageManagerNPM,
		TTY:            false,
		CI:             true,
	}

	err := svc.ensureIgnoreFiles(ctx, ws)
	if err == nil {
		t.Fatal("expected error when .gitignore cannot be read, got nil")
	}

	expectedMsg := "permission denied"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error to contain %q, got: %v", expectedMsg, err)
	}
}

func TestEnsureNPMHardening_PropagatesReadErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, fmt.Errorf("read failed")
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, os.ErrNotExist
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			return nil
		},
	}

	svc := NewService(fakeFS, nil, nil, nil, nil)

	err := svc.ensureNPMHardening(ctx, "/root/.npmrc")
	if err == nil {
		t.Fatal("expected error when .npmrc cannot be read, got nil")
	}

	expectedMsg := "read failed"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error to contain %q, got: %v", expectedMsg, err)
	}
}

func TestEnsureBunHardening_PropagatesReadErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, fmt.Errorf("read failed")
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, os.ErrNotExist
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			return nil
		},
	}

	svc := NewService(fakeFS, nil, nil, nil, nil)

	err := svc.ensureBunHardening(ctx, "/root/bunfig.toml")
	if err == nil {
		t.Fatal("expected error when bunfig.toml cannot be read, got nil")
	}

	expectedMsg := "read failed"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error to contain %q, got: %v", expectedMsg, err)
	}
}

func TestEnsureDenoHardening_PropagatesReadErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, fmt.Errorf("read failed")
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, os.ErrNotExist
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			return nil
		},
	}

	svc := NewService(fakeFS, nil, nil, nil, nil)

	err := svc.ensureDenoHardening(ctx, "/root/deno.json")
	if err == nil {
		t.Fatal("expected error when deno.json cannot be read, got nil")
	}

	expectedMsg := "read failed"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error to contain %q, got: %v", expectedMsg, err)
	}
}

func TestEnsureIgnoreFiles_CreatesWhenNotExists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writtenFiles := map[string][]byte{}

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, os.ErrNotExist
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, os.ErrNotExist
		},
		WriteFileFn: func(ctx context.Context, path string, data []byte) error {
			writtenFiles[path] = data
			return nil
		},
	}

	svc := NewService(fakeFS, nil, nil, nil, nil)
	ws := shared.Workspace{
		Root:           "/root",
		ManifestPath:   "/root/package.json",
		PackageManager: shared.PackageManagerNPM,
		TTY:            false,
		CI:             true,
	}

	err := svc.ensureIgnoreFiles(ctx, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both files should have been created
	if len(writtenFiles) != 2 {
		t.Errorf("expected 2 files to be written, got %d", len(writtenFiles))
	}

	// .gitignore should contain required entries
	gitignore := string(writtenFiles["/root/.gitignore"])
	if !strings.Contains(gitignore, ".env.local") {
		t.Error("expected .gitignore to contain .env.local")
	}
	if !strings.Contains(gitignore, "coverage/") {
		t.Error("expected .gitignore to contain coverage/")
	}
}
