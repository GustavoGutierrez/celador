package fs

import (
	"context"
	"fmt"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/ports"
)

const (
	managedStart = "<!-- celador:start -->"
	managedEnd   = "<!-- celador:end -->"
)

func WriteManagedBlock(ctx context.Context, fs ports.FileSystem, path string, block string) error {
	return WriteManagedSection(ctx, fs, path, managedStart, managedEnd, block)
}

func WriteManagedSection(ctx context.Context, fs ports.FileSystem, path string, startMarker string, endMarker string, block string) error {
	existing, _ := fs.ReadFile(ctx, path)
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
