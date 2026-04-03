package install

import (
	"context"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestAssessAndExecute(t *testing.T) {
	t.Parallel()
	pm := &helpers.StubPM{}
	service := NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: "/tmp/project", PackageManager: shared.PackageManagerNPM}},
		helpers.StubMetadata{Assessment: shared.InstallAssessment{Package: "left-pad", Risk: shared.SeverityLow}},
		pm,
		&helpers.StubUI{},
	)
	assessment, err := service.Assess(context.Background(), "/tmp/project", false, true, []string{"left-pad"})
	if err != nil {
		t.Fatalf("assess: %v", err)
	}
	if assessment.Package != "left-pad" {
		t.Fatalf("unexpected assessment: %+v", assessment)
	}
	if err := service.Execute(context.Background(), "/tmp/project", false, true, []string{"left-pad"}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(pm.Calls) != 1 {
		t.Fatalf("expected package manager invocation")
	}
}
