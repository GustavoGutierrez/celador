package fs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteManagedBlockPreservesUserContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := NewOSFileSystem(root)
	path := filepath.Join(root, "AGENTS.md")
	original := "# Custom intro\n\n<!-- celador:start -->\nold\n<!-- celador:end -->\n\n# Footer\n"
	if err := fs.WriteFile(context.Background(), path, []byte(original)); err != nil {
		t.Fatalf("write original: %v", err)
	}
	if err := WriteManagedBlock(context.Background(), fs, path, "<!-- celador:start -->\nnew\n<!-- celador:end -->"); err != nil {
		t.Fatalf("write managed block: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "# Custom intro") || !strings.Contains(text, "# Footer") || !strings.Contains(text, "new") {
		t.Fatalf("managed block update lost content: %s", text)
	}
}

func TestWriteManagedLLMPreservesUserContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fs := NewOSFileSystem(root)
	path := filepath.Join(root, "llm.txt")
	original := "Project-specific notes\n\n# celador:start\nold\n# celador:end\n"
	if err := fs.WriteFile(context.Background(), path, []byte(original)); err != nil {
		t.Fatalf("write original: %v", err)
	}
	if err := WriteManagedLLM(context.Background(), fs, path, "# Celador LLM Guide\nnew"); err != nil {
		t.Fatalf("write managed llm: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "Project-specific notes") || !strings.Contains(text, "# Celador LLM Guide") || !strings.Contains(text, "new") {
		t.Fatalf("managed llm update lost content: %s", text)
	}
}
