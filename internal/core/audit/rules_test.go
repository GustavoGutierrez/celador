package audit

import (
	"context"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestRuleEvaluator_Evaluate_NoRules(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFS := &helpers.FakeFileSystem{
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	eval := NewRuleEvaluator(fakeFS)
	workspace := shared.Workspace{Root: "/root"}
	findings, err := eval.Evaluate(ctx, workspace, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty rules, got %d", len(findings))
	}
}

func TestRuleEvaluator_Evaluate_MustNotFind(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fileContent := `const query = "SELECT * FROM users WHERE id=" + userInput;`
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(fileContent), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	eval := NewRuleEvaluator(fakeFS)
	workspace := shared.Workspace{Root: "/root"}
	rules := []shared.RuleConfig{{
		ID:          "no-sql-injection",
		Frameworks:  []string{},
		File:        "src/db.ts",
		MustNotFind: "SELECT * FROM",
		Summary:     "SQL injection detected",
		Severity:    shared.SeverityHigh,
	}}

	findings, err := eval.Evaluate(ctx, workspace, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].ID != "no-sql-injection" {
		t.Errorf("expected ID 'no-sql-injection', got %q", findings[0].ID)
	}
	if findings[0].Severity != shared.SeverityHigh {
		t.Errorf("expected severity high, got %v", findings[0].Severity)
	}
}

func TestRuleEvaluator_Evaluate_FrameworkMismatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fileContent := `some dangerous code`
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(fileContent), nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	eval := NewRuleEvaluator(fakeFS)
	workspace := shared.Workspace{Root: "/root", Frameworks: []string{"react"}}
	rules := []shared.RuleConfig{{
		ID:          "vue-rule",
		Frameworks:  []string{"vue"},
		File:        "src/App.vue",
		MustNotFind: "dangerous",
		Summary:     "Vue-specific rule",
		Severity:    shared.SeverityMedium,
	}}

	findings, err := eval.Evaluate(ctx, workspace, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings (wrong framework), got %d", len(findings))
	}
}

func TestScanTailwindArbitraryValues_DynamicClasses(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fileContent := `const cls = "bg-[" + userInput + "]";`
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(fileContent), nil
		},
		WalkFilesFn: func(ctx context.Context, root string, exts []string) ([]string, error) {
			return []string{"/root/src/App.tsx"}, nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	eval := NewRuleEvaluator(fakeFS)
	workspace := shared.Workspace{Root: "/root"}
	findings, err := eval.scanTailwindArbitraryValues(ctx, workspace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for dynamic arbitrary value, got %d", len(findings))
	}
	if findings[0].RuleID != "tailwind-dynamic-arbitrary-value" {
		t.Errorf("expected rule ID 'tailwind-dynamic-arbitrary-value', got %q", findings[0].RuleID)
	}
}

func TestScanTailwindArbitraryValues_StaticClasses_NotFlagged(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fileContent := `<div className="bg-red-500 text-center">Hello</div>`
	fakeFS := &helpers.FakeFileSystem{
		ReadFileFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(fileContent), nil
		},
		WalkFilesFn: func(ctx context.Context, root string, exts []string) ([]string, error) {
			return []string{"/root/src/App.tsx"}, nil
		},
		StatFn: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	eval := NewRuleEvaluator(fakeFS)
	workspace := shared.Workspace{Root: "/root"}
	findings, err := eval.scanTailwindArbitraryValues(ctx, workspace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for static classes, got %d", len(findings))
	}
}

func TestMatchesFramework(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ws       []string
		rules    []string
		expected bool
	}{
		{"no_rule_frameworks", []string{"react"}, []string{}, true},
		{"matching_framework", []string{"react"}, []string{"react"}, true},
		{"multiple_frameworks_match", []string{"react", "next"}, []string{"next"}, true},
		{"no_match", []string{"react"}, []string{"vue"}, false},
		{"empty_workspace", []string{}, []string{"react"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesFramework(tt.ws, tt.rules)
			if result != tt.expected {
				t.Errorf("matchesFramework(%v, %v) = %v, want %v", tt.ws, tt.rules, result, tt.expected)
			}
		})
	}
}
