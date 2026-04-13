package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNodeVersionDetector_Detect_ValidOutput(t *testing.T) {
	dir := t.TempDir()
	createMockNode(t, dir, "v20.11.1")

	detector := NewNodeVersionDetector()
	version, ok := detector.Detect(context.Background(), dir)
	if !ok {
		t.Fatal("expected version detection to succeed")
	}
	if version != "20.11.1" {
		t.Errorf("expected version '20.11.1', got %q", version)
	}
}

func TestNodeVersionDetector_Detect_NoNodeInstalled(t *testing.T) {
	dir := t.TempDir()

	// Override PATH to an empty directory that has no `node` binary
	emptyDir := t.TempDir()
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", emptyDir)
	defer os.Setenv("PATH", originalPath)

	detector := NewNodeVersionDetector()
	_, ok := detector.Detect(context.Background(), dir)
	if ok {
		t.Error("expected version detection to fail when node is not installed")
	}
}

func TestNodeVersionDetector_Detect_UnexpectedFormat(t *testing.T) {
	dir := t.TempDir()
	createMockNode(t, dir, "not-a-version-string")

	detector := NewNodeVersionDetector()
	_, ok := detector.Detect(context.Background(), dir)
	if ok {
		t.Error("expected version detection to fail for non-semver output")
	}
}

func TestNodeVersionPattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantOk  bool
		wantVer string
	}{
		{"standard", "v20.11.1", true, "20.11.1"},
		{"no_v_prefix", "18.17.0", true, "18.17.0"},
		{"prerelease", "21.0.0-rc.1", true, "21.0.0-rc.1"},
		{"build_metadata", "16.13.0+build.123", true, "16.13.0+build.123"},
		{"empty", "", false, ""},
		{"garbage", "not-a-version", false, ""},
		{"partial", "v20", false, ""},
		{"extra_text", "v20.11.1 extra", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := nodeVersionPattern.FindStringSubmatch(tt.input)
			ok := len(match) == 2
			if ok != tt.wantOk {
				t.Errorf("match %v, want %v for input %q", ok, tt.wantOk, tt.input)
			}
			if ok && match[1] != tt.wantVer {
				t.Errorf("version %q, want %q", match[1], tt.wantVer)
			}
		})
	}
}

// createMockNode creates a fake `node` script in the given directory
// that outputs the specified version when called with `--version`.
func createMockNode(t *testing.T, dir string, versionOutput string) {
	t.Helper()

	scriptPath := filepath.Join(dir, "node")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo \"" + versionOutput + "\"\n" +
		"fi\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock node script: %v", err)
	}

	// Set PATH to only include our mock directory, so the test
	// uses the fake `node` instead of the system binary.
	originalPath := os.Getenv("PATH")
	t.Cleanup(func() {
		os.Setenv("PATH", originalPath)
	})
	os.Setenv("PATH", dir)
}

func TestNodeVersionDetector_ContextCancelled(t *testing.T) {
	dir := t.TempDir()
	// Create a mock node that sleeps forever to simulate a hung process
	scriptPath := filepath.Join(dir, "node")
	script := "#!/bin/sh\nsleep 10\n"
	os.WriteFile(scriptPath, []byte(script), 0755)

	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	os.Setenv("PATH", dir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	detector := NewNodeVersionDetector()
	_, ok := detector.Detect(ctx, dir)
	if ok {
		t.Error("expected version detection to fail when context is cancelled")
	}
}
