package install

import (
	"context"
	"fmt"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type Service struct {
	detector ports.WorkspaceDetector
	meta     ports.PackageMetadataSource
	pm       ports.PackageManager
	ui       ports.PromptUI
}

func NewService(detector ports.WorkspaceDetector, meta ports.PackageMetadataSource, pm ports.PackageManager, ui ports.PromptUI) *Service {
	return &Service{detector: detector, meta: meta, pm: pm, ui: ui}
}

func (s *Service) Assess(ctx context.Context, root string, tty bool, ci bool, args []string) (shared.InstallAssessment, error) {
	ws, err := s.detector.Detect(ctx, root, tty, ci)
	if err != nil {
		return shared.InstallAssessment{}, err
	}
	assessment, err := s.meta.InspectPackage(ctx, ws.PackageManager, args[0])
	if err != nil {
		return shared.InstallAssessment{}, fmt.Errorf("inspect package: %w", err)
	}
	assessment.Manager = ws.PackageManager
	return assessment, nil
}

func (s *Service) Execute(ctx context.Context, root string, tty bool, ci bool, args []string) error {
	ws, err := s.detector.Detect(ctx, root, tty, ci)
	if err != nil {
		return err
	}
	return s.pm.Install(ctx, ws, args)
}
