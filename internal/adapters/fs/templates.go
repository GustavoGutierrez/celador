package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/ports"
)

const (
	managedStart = "<!-- celador:start -->"
	managedEnd   = "<!-- celador:end -->"
)

// ValidateWorkspacePath ensures the given path is within the workspace root.
// Returns an error if the path attempts to escape the workspace directory.
func ValidateWorkspacePath(path string, workspaceRoot string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return fmt.Errorf("resolve workspace root: %w", err)
	}
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) && absPath != absRoot {
		return fmt.Errorf("path %q is outside workspace root %q", path, workspaceRoot)
	}
	return nil
}

func WriteManagedBlock(ctx context.Context, fs ports.FileSystem, path string, block string) error {
	return WriteManagedSection(ctx, fs, path, managedStart, managedEnd, block)
}

func WriteManagedSection(ctx context.Context, fs ports.FileSystem, path string, startMarker string, endMarker string, block string) error {
	existing, err := fs.ReadFile(ctx, path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read %s: %w", path, err)
		}
		existing = []byte{}
	}
	content := string(existing)
	if strings.Contains(content, startMarker) && strings.Contains(content, endMarker) {
		start := strings.Index(content, startMarker)
		end := strings.Index(content, endMarker) + len(endMarker)
		content = content[:start] + strings.TrimSpace(block) + "\n" + content[end:]
	} else if strings.TrimSpace(content) == "" {
		content = strings.TrimSpace(block) + "\n"
	} else {
		content = strings.TrimRight(content, "\n") + "\n\n" + strings.TrimSpace(block) + "\n"
	}
	if err := fs.WriteFile(ctx, path, []byte(content)); err != nil {
		return fmt.Errorf("write managed block %s: %w", path, err)
	}
	return nil
}

func WriteIfMissing(ctx context.Context, fs ports.FileSystem, path string, data []byte) error {
	if ok, err := fs.Stat(ctx, path); err != nil {
		return err
	} else if ok {
		return nil
	}
	return fs.WriteFile(ctx, path, data)
}
