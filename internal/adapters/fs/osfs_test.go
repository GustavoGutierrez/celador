package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOSFileSystem_ReadWriteRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fs := NewOSFileSystem(dir)
	ctx := context.Background()

	path := filepath.Join(dir, "test.txt")
	data := []byte("hello world")

	err := fs.WriteFile(ctx, path, data)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	result, err := fs.ReadFile(ctx, path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(result) != string(data) {
		t.Errorf("expected %q, got %q", data, result)
	}
}

func TestOSFileSystem_Stat_FileExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fs := NewOSFileSystem(dir)
	ctx := context.Background()

	path := filepath.Join(dir, "exists.txt")
	os.WriteFile(path, []byte("test"), 0644)

	exists, err := fs.Stat(ctx, path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}
}

func TestOSFileSystem_Stat_FileNotExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fs := NewOSFileSystem(dir)
	ctx := context.Background()

	exists, err := fs.Stat(ctx, filepath.Join(dir, "missing.txt"))
	if err != nil {
		t.Fatalf("stat should not error for missing file: %v", err)
	}
	if exists {
		t.Error("expected file to not exist")
	}
}

func TestOSFileSystem_MkdirAll_CreatesNestedDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fs := NewOSFileSystem(dir)
	ctx := context.Background()

	nestedPath := filepath.Join(dir, "a", "b", "c")
	err := fs.MkdirAll(ctx, nestedPath)
	if err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	info, err := os.Stat(nestedPath)
	if err != nil {
		t.Fatalf("stat on created dir failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestOSFileSystem_Glob_MatchesPattern(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fs := NewOSFileSystem(dir)
	ctx := context.Background()

	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "file3.js"), []byte("c"), 0644)

	matches, err := fs.Glob(ctx, dir, "*.txt")
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestOSFileSystem_WalkFiles_FiltersByExtension(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fs := NewOSFileSystem(dir)
	ctx := context.Background()

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "c.js"), []byte("c"), 0644)

	files, err := fs.WalkFiles(ctx, dir, []string{".txt"})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 .txt files, got %d", len(files))
	}
	for _, f := range files {
		if filepath.Ext(f) != ".txt" {
			t.Errorf("expected .txt extension, got %s", f)
		}
	}
}

func TestOSFileSystem_ExecRoot_ReturnsRoot(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fs := NewOSFileSystem(dir)

	if fs.ExecRoot() != dir {
		t.Errorf("expected root %q, got %q", dir, fs.ExecRoot())
	}
}
