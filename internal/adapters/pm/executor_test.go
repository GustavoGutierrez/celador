package pm

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

// mockPM creates a fake package manager script in the temp directory
// that echoes its arguments to stdout for verification.
func mockPM(t *testing.T, dir string, name string) {
	t.Helper()
	script := filepath.Join(dir, name)
	content := "#!/bin/sh\necho \"$0\" \"$@\"\n"
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write mock %s: %v", name, err)
	}
}

func TestExecutor_Install_NPM(t *testing.T) {
	dir := t.TempDir()
	mockPM(t, dir, "npm")

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", originalPath)

	var stdout, stderr bytes.Buffer
	executor := NewExecutor(&stdout, &stderr)
	workspace := shared.Workspace{
		Root:           dir,
		PackageManager: shared.PackageManagerNPM,
	}

	err := executor.Install(context.Background(), workspace, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := stdout.String()
	if output == "" {
		t.Fatal("expected output from mock npm, got empty string")
	}
	// Verify npm was called with "install" (path may vary)
	if !strings.Contains(output, "npm") || !strings.Contains(output, "install") {
		t.Errorf("expected output to contain 'npm install', got %q", output)
	}
}

func TestExecutor_Install_PNPM(t *testing.T) {
	dir := t.TempDir()
	mockPM(t, dir, "pnpm")

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", originalPath)

	var stdout, stderr bytes.Buffer
	executor := NewExecutor(&stdout, &stderr)
	workspace := shared.Workspace{
		Root:           dir,
		PackageManager: shared.PackageManagerPNPM,
	}

	err := executor.Install(context.Background(), workspace, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "pnpm") || !strings.Contains(output, "install") {
		t.Errorf("expected 'pnpm install', got %q", output)
	}
}

func TestExecutor_Install_Bun(t *testing.T) {
	dir := t.TempDir()
	mockPM(t, dir, "bun")

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", originalPath)

	var stdout, stderr bytes.Buffer
	executor := NewExecutor(&stdout, &stderr)
	workspace := shared.Workspace{
		Root:           dir,
		PackageManager: shared.PackageManagerBun,
	}

	err := executor.Install(context.Background(), workspace, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := stdout.String()
	// Bun uses "add" instead of "install"
	if !strings.Contains(output, "bun") || !strings.Contains(output, "add") {
		t.Errorf("expected 'bun add', got %q", output)
	}
}

func TestExecutor_Install_WithArgs(t *testing.T) {
	dir := t.TempDir()
	mockPM(t, dir, "npm")

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", originalPath)

	var stdout bytes.Buffer
	executor := NewExecutor(&stdout, os.Stderr)
	workspace := shared.Workspace{
		Root:           dir,
		PackageManager: shared.PackageManagerNPM,
	}

	// Install specific packages
	err := executor.Install(context.Background(), workspace, []string{"express", "lodash"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "npm install express lodash") {
		t.Errorf("expected 'npm install express lodash' in output, got %q", output)
	}
}

func TestExecutor_Install_UnknownManager(t *testing.T) {
	t.Parallel()
	executor := NewExecutor(os.Stdout, os.Stderr)
	workspace := shared.Workspace{
		Root:           t.TempDir(),
		PackageManager: shared.PackageManagerUnknown,
	}

	err := executor.Install(context.Background(), workspace, []string{"pkg"})
	if err == nil {
		t.Fatal("expected error for unknown package manager, got nil")
	}
}

func TestExecutor_Install_BinaryMissing_ReturnsError(t *testing.T) {
	dir := t.TempDir()

	// Set PATH to empty dir - no npm/pnpm/bun available
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", originalPath)

	var stdout, stderr bytes.Buffer
	executor := NewExecutor(&stdout, &stderr)
	workspace := shared.Workspace{
		Root:           dir,
		PackageManager: shared.PackageManagerNPM,
	}

	err := executor.Install(context.Background(), workspace, nil)
	if err == nil {
		t.Fatal("expected error when npm binary is missing, got nil")
	}
}

func TestExecutor_Install_ContextCancelled(t *testing.T) {
	dir := t.TempDir()
	// Create a mock npm that sleeps forever
	script := filepath.Join(dir, "npm")
	os.WriteFile(script, []byte("#!/bin/sh\nsleep 10\n"), 0755)

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", originalPath)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var stdout, stderr bytes.Buffer
	executor := NewExecutor(&stdout, &stderr)
	workspace := shared.Workspace{
		Root:           dir,
		PackageManager: shared.PackageManagerNPM,
	}

	err := executor.Install(ctx, workspace, nil)
	if err == nil {
		t.Fatal("expected error when context is cancelled, got nil")
	}
}

func TestExecutor_CapturesStderr(t *testing.T) {
	dir := t.TempDir()
	// Create a mock npm that writes to stderr
	script := filepath.Join(dir, "npm")
	os.WriteFile(script, []byte("#!/bin/sh\necho 'error: something failed' >&2\nexit 1\n"), 0755)

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", originalPath)

	var stdout, stderr bytes.Buffer
	executor := NewExecutor(&stdout, &stderr)
	workspace := shared.Workspace{
		Root:           dir,
		PackageManager: shared.PackageManagerNPM,
	}

	err := executor.Install(context.Background(), workspace, nil)
	if err == nil {
		t.Fatal("expected error from failing npm command, got nil")
	}
	if stderr.String() == "" {
		t.Error("expected stderr to be captured, got empty string")
	}
}
