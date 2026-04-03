package audit

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type RuleEvaluator struct{ fs ports.FileSystem }

func NewRuleEvaluator(fs ports.FileSystem) *RuleEvaluator { return &RuleEvaluator{fs: fs} }

func (e *RuleEvaluator) Evaluate(ctx context.Context, workspace shared.Workspace, rules []shared.RuleConfig) ([]shared.Finding, error) {
	findings := []shared.Finding{}
	for _, rule := range rules {
		if !matchesFramework(workspace.Frameworks, rule.Frameworks) {
			continue
		}
		path := filepath.Join(workspace.Root, rule.File)
		body, err := e.fs.ReadFile(ctx, path)
		if err != nil {
			continue
		}
		text := string(body)
		if rule.MustContain != "" && !strings.Contains(text, rule.MustContain) {
			findings = append(findings, shared.Finding{ID: rule.ID, Source: shared.FindingSourceRule, Severity: rule.Severity, Target: rule.File, RuleID: rule.ID, Summary: rule.Summary, Locations: []shared.FindingLocation{{Path: rule.File, Line: 1}}})
		}
		if rule.MustNotFind != "" && strings.Contains(text, rule.MustNotFind) {
			findings = append(findings, shared.Finding{ID: rule.ID, Source: shared.FindingSourceRule, Severity: rule.Severity, Target: rule.File, RuleID: rule.ID, Summary: rule.Summary, Locations: []shared.FindingLocation{{Path: rule.File, Line: 1}}})
		}
	}
	tailwind, err := e.scanTailwindArbitraryValues(ctx, workspace)
	if err != nil {
		return nil, err
	}
	findings = append(findings, tailwind...)
	return findings, nil
}

func (e *RuleEvaluator) scanTailwindArbitraryValues(ctx context.Context, workspace shared.Workspace) ([]shared.Finding, error) {
	files, err := e.fs.WalkFiles(ctx, workspace.Root, []string{".tsx", ".vue", ".svelte"})
	if err != nil {
		return nil, fmt.Errorf("walk tailwind source files: %w", err)
	}
	findings := []shared.Finding{}
	for _, path := range files {
		body, err := e.fs.ReadFile(ctx, path)
		if err != nil {
			return nil, err
		}
		text := string(body)
		if strings.Contains(text, "bg-[") && strings.Contains(text, "+") {
			findings = append(findings, shared.Finding{ID: "tailwind-dynamic-arbitrary-value", Source: shared.FindingSourceRule, Severity: shared.SeverityHigh, Target: path, RuleID: "tailwind-dynamic-arbitrary-value", Summary: "Tailwind arbitrary value uses string interpolation", Locations: []shared.FindingLocation{{Path: path, Line: 1}}})
		}
	}
	return findings, nil
}

func matchesFramework(workspaceFrameworks []string, ruleFrameworks []string) bool {
	if len(ruleFrameworks) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, framework := range workspaceFrameworks {
		set[framework] = struct{}{}
	}
	for _, framework := range ruleFrameworks {
		if _, ok := set[framework]; ok {
			return true
		}
	}
	return false
}
