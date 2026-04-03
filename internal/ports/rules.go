package ports

import (
	"context"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type RuleLoader interface {
	Load(ctx context.Context, root string) ([]shared.RuleConfig, string, error)
}

type RuleEvaluator interface {
	Evaluate(ctx context.Context, workspace shared.Workspace, rules []shared.RuleConfig) ([]shared.Finding, error)
}

type IgnoreStore interface {
	Load(ctx context.Context, root string) ([]shared.IgnoreRule, error)
}

type PatchWriter interface {
	Preview(ctx context.Context, workspace shared.Workspace, plan shared.FixPlan) (string, error)
	Apply(ctx context.Context, workspace shared.Workspace, plan shared.FixPlan) error
}
