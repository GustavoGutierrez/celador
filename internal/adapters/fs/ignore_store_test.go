package fs

import (
	"context"
	"os"
	"testing"

	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestIgnoreStore_Load_WithRules(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	content := "GHSA-xxxx|accepted temporarily|2026-12-31\nnext.config.js|legacy exception|\n"
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(content), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	store := NewIgnoreStore(fakeFS)
	rules, err := store.Load(ctx, "/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Selector != "GHSA-xxxx" {
		t.Errorf("expected selector 'GHSA-xxxx', got %q", rules[0].Selector)
	}
	if rules[0].Reason != "accepted temporarily" {
		t.Errorf("expected reason 'accepted temporarily', got %q", rules[0].Reason)
	}
	if rules[0].ExpiresAt == nil || rules[0].ExpiresAt.Year() != 2026 {
		t.Errorf("expected expiry 2026-12-31, got %v", rules[0].ExpiresAt)
	}
	if rules[1].Selector != "next.config.js" {
		t.Errorf("expected selector 'next.config.js', got %q", rules[1].Selector)
	}
}

func TestIgnoreStore_Load_MissingFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, os.ErrNotExist
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	store := NewIgnoreStore(fakeFS)
	rules, err := store.Load(ctx, "/root")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if rules != nil {
		t.Errorf("expected nil rules for missing file, got %v", rules)
	}
}

func TestIgnoreStore_Load_SkipsCommentsAndEmptyLines(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	content := "# This is a comment\n\nGHSA-active|valid rule|\n\n# Another comment\n"
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(content), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	store := NewIgnoreStore(fakeFS)
	rules, err := store.Load(ctx, "/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (comments and empty lines skipped), got %d", len(rules))
	}
	if rules[0].Selector != "GHSA-active" {
		t.Errorf("expected 'GHSA-active', got %q", rules[0].Selector)
	}
}

func TestIgnoreStore_Load_InvalidDate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	content := "GHSA-bad-date|rule with bad date|not-a-date\n"
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(content), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	store := NewIgnoreStore(fakeFS)
	rules, err := store.Load(ctx, "/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Selector != "GHSA-bad-date" {
		t.Errorf("expected selector 'GHSA-bad-date', got %q", rules[0].Selector)
	}
	if rules[0].ExpiresAt != nil {
		t.Errorf("expected nil ExpiresAt for invalid date, got %v", rules[0].ExpiresAt)
	}
}
