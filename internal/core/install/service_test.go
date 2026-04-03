package install

import (
	"context"
	"path/filepath"
	"testing"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	workspacecore "github.com/GustavoGutierrez/celador/internal/core/workspace"
	"github.com/GustavoGutierrez/celador/test/helpers"
)

func TestAssessAndExecute(t *testing.T) {
	t.Parallel()
	pm := &helpers.StubPM{}
	service := NewService(
		helpers.StubDetector{Workspace: shared.Workspace{Root: "/tmp/project", PackageManager: shared.PackageManagerNPM}},
		&helpers.StubMetadata{Assessment: shared.InstallAssessment{Package: "left-pad", Risk: shared.SeverityLow}},
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

func TestAssessAndExecuteInferNPMWithoutLockfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fs := fsadapter.NewOSFileSystem(root)
	ctx := context.Background()
	if err := fs.WriteFile(ctx, filepath.Join(root, "package.json"), []byte(`{"name":"demo"}`)); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	pm := &helpers.StubPM{}
	meta := &helpers.StubMetadata{Assessment: shared.InstallAssessment{Package: "express", Risk: shared.SeverityLow}}
	service := NewService(
		workspacecore.NewDetector(fs),
		meta,
		pm,
		&helpers.StubUI{},
	)

	assessment, err := service.Assess(ctx, root, false, true, []string{"express"})
	if err != nil {
		t.Fatalf("assess: %v", err)
	}
	if assessment.Manager != shared.PackageManagerNPM {
		t.Fatalf("expected npm assessment manager, got %s", assessment.Manager)
	}
	if meta.LastManager != shared.PackageManagerNPM {
		t.Fatalf("expected metadata inspection to use npm, got %s", meta.LastManager)
	}

	if err := service.Execute(ctx, root, false, true, []string{"express"}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(pm.Workspaces) != 1 {
		t.Fatalf("expected one install execution, got %d", len(pm.Workspaces))
	}
	if pm.Workspaces[0].PackageManager != shared.PackageManagerNPM {
		t.Fatalf("expected install execution to use npm, got %s", pm.Workspaces[0].PackageManager)
	}
}
