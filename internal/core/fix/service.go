package fix

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/audit"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type Service struct {
	scan    *audit.Service
	patches ports.PatchWriter
	fs      ports.FileSystem
	ui      ports.PromptUI
}

func NewService(scan *audit.Service, patches ports.PatchWriter, fs ports.FileSystem, ui ports.PromptUI) *Service {
	return &Service{scan: scan, patches: patches, fs: fs, ui: ui}
}

func (s *Service) Plan(ctx context.Context, root string, tty bool, ci bool) (shared.FixPlan, error) {
	result, err := s.scan.Run(ctx, root, tty, ci)
	if err != nil {
		return shared.FixPlan{}, err
	}
	manifestDeps, err := s.readManifestDependencies(ctx, result.Workspace.ManifestPath)
	if err != nil {
		return shared.FixPlan{}, err
	}
	ops := []shared.FixOperation{}
	for _, finding := range result.Findings {
		if !finding.Fixable || finding.PackageName == "" || finding.FixVersion == "" {
			continue
		}
		current := manifestDeps[finding.PackageName]
		strategy := "override"
		if current != "" {
			strategy = "bump"
		}
		ops = append(ops, shared.FixOperation{File: result.Workspace.ManifestPath, PackageName: finding.PackageName, CurrentVersion: current, ProposedVersion: finding.FixVersion, Strategy: strategy, BlastRadius: []string{finding.PackageName}, RequiresInstall: true})
	}
	unique := dedupeOperations(ops)
	plan := shared.FixPlan{Operations: unique, Summary: fmt.Sprintf("Planned %d conservative remediation operations", len(unique))}
	plan.DryRunDiff = fsadapter.RenderPlanDiff(unique)
	return plan, nil
}

func (s *Service) Apply(ctx context.Context, root string, tty bool, ci bool, plan shared.FixPlan) error {
	ws := shared.Workspace{Root: root, ManifestPath: root + "/package.json", TTY: tty, CI: ci}
	return s.patches.Apply(ctx, ws, plan)
}

func (s *Service) readManifestDependencies(ctx context.Context, manifestPath string) (map[string]string, error) {
	body, err := s.fs.ReadFile(ctx, manifestPath)
	if err != nil {
		return nil, err
	}
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(body, &pkg); err != nil {
		return nil, err
	}
	return pkg.Dependencies, nil
}

func dedupeOperations(ops []shared.FixOperation) []shared.FixOperation {
	byPackage := map[string]shared.FixOperation{}
	for _, op := range ops {
		existing, ok := byPackage[op.PackageName]
		if !ok || existing.ProposedVersion < op.ProposedVersion {
			byPackage[op.PackageName] = op
		}
	}
	result := make([]shared.FixOperation, 0, len(byPackage))
	for _, op := range byPackage {
		result = append(result, op)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PackageName < result[j].PackageName })
	return result
}
