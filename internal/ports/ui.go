package ports

import (
	"context"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type PromptUI interface {
	Confirm(ctx context.Context, prompt string) (bool, error)
	RenderScan(ctx context.Context, result shared.ScanResult, options shared.ScanRenderOptions) error
	RenderFixPlan(ctx context.Context, plan shared.FixPlan) error
	RenderInstallAssessment(ctx context.Context, assessment shared.InstallAssessment) error
	RenderOverview(ctx context.Context, overview shared.Overview, interactive bool) error
	Printf(format string, args ...any)
}
