package ports

import (
	"context"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type VulnerabilitySource interface {
	Query(ctx context.Context, deps []shared.Dependency) ([]shared.Finding, error)
}

type PackageMetadataSource interface {
	InspectPackage(ctx context.Context, manager shared.PackageManager, pkg string) (shared.InstallAssessment, error)
}
