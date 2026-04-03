package ports

import (
	"context"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type PackageManager interface {
	Install(ctx context.Context, workspace shared.Workspace, args []string) error
}

type WorkspaceDetector interface {
	Detect(ctx context.Context, root string, tty bool, ci bool) (shared.Workspace, error)
}

type LockfileParser interface {
	Supports(path string) bool
	Parse(ctx context.Context, workspace shared.Workspace, path string) ([]shared.Dependency, error)
}
