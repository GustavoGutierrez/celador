package fix

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
	findings := shared.RenderedFindings(result.Findings)
	manifestDeps, err := s.readManifestDependencies(ctx, result.Workspace.ManifestPath)
	if err != nil {
		return shared.FixPlan{}, err
	}
	ops := []shared.FixOperation{}
	reasons := newReasonAccumulator()
	for _, finding := range findings {
		if reason, ok := classifyUnplannedFinding(finding); ok {
			reasons.Add(reason, formatFindingExample(finding))
			continue
		}
		current, found := manifestDeps[finding.PackageName]
		strategy := "override"
		section := ""
		currentVersion := ""
		if found {
			strategy = "bump"
			section = current.Section
			currentVersion = current.Version
		}
		if reason, ok := classifyNonConservativeUpgrade(currentVersion, finding.FixVersion); ok {
			reasons.Add(reason, formatFindingExample(finding))
			continue
		}
		ops = append(ops, shared.FixOperation{
			File:            result.Workspace.ManifestPath,
			ManifestSection: section,
			PackageName:     finding.PackageName,
			CurrentVersion:  currentVersion,
			ProposedVersion: finding.FixVersion,
			Strategy:        strategy,
			BlastRadius:     []string{finding.PackageName},
			RequiresInstall: true,
		})
	}
	unique := dedupeOperations(ops)
	plan := shared.FixPlan{
		Operations: unique,
		Summary:    fmt.Sprintf("Planned %d conservative remediation operations", len(unique)),
		Reasons:    reasons.List(),
	}
	plan.DryRunDiff = fsadapter.RenderPlanDiff(unique)
	return plan, nil
}

func (s *Service) Apply(ctx context.Context, root string, tty bool, ci bool, plan shared.FixPlan) error {
	ws := shared.Workspace{Root: root, ManifestPath: root + "/package.json", TTY: tty, CI: ci}
	return s.patches.Apply(ctx, ws, plan)
}

type manifestDependency struct {
	Section string
	Version string
}

func (s *Service) readManifestDependencies(ctx context.Context, manifestPath string) (map[string]manifestDependency, error) {
	body, err := s.fs.ReadFile(ctx, manifestPath)
	if err != nil {
		return nil, err
	}
	var pkg struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
		PeerDependencies     map[string]string `json:"peerDependencies"`
	}
	if err := json.Unmarshal(body, &pkg); err != nil {
		return nil, err
	}
	deps := map[string]manifestDependency{}
	mergeManifestDeps(deps, "dependencies", pkg.Dependencies)
	mergeManifestDeps(deps, "devDependencies", pkg.DevDependencies)
	mergeManifestDeps(deps, "optionalDependencies", pkg.OptionalDependencies)
	mergeManifestDeps(deps, "peerDependencies", pkg.PeerDependencies)
	return deps, nil
}

func mergeManifestDeps(target map[string]manifestDependency, section string, deps map[string]string) {
	for name, version := range deps {
		if _, exists := target[name]; exists {
			continue
		}
		target[name] = manifestDependency{Section: section, Version: version}
	}
}

func classifyUnplannedFinding(finding shared.Finding) (shared.FixPlanReasonCategory, bool) {
	if finding.Source != shared.FindingSourceOSV {
		return shared.FixPlanReasonManualChange, true
	}
	if strings.TrimSpace(finding.FixVersion) == "" || !finding.Fixable {
		return shared.FixPlanReasonNoFixedVersion, true
	}
	if strings.TrimSpace(finding.PackageName) == "" {
		return shared.FixPlanReasonOutsideScope, true
	}
	return "", false
}

func classifyNonConservativeUpgrade(currentVersion string, proposedVersion string) (shared.FixPlanReasonCategory, bool) {
	if shared.NormalizeVersion(proposedVersion) == "" {
		return shared.FixPlanReasonNonConservativeUpgrade, true
	}
	if shared.IsPrereleaseVersion(proposedVersion) {
		return shared.FixPlanReasonNonConservativeUpgrade, true
	}
	if strings.TrimSpace(currentVersion) == "" {
		return "", false
	}
	currentMajor := shared.VersionMajor(currentVersion)
	proposedMajor := shared.VersionMajor(proposedVersion)
	if currentMajor == "" || proposedMajor == "" {
		return shared.FixPlanReasonNonConservativeUpgrade, true
	}
	if currentMajor != proposedMajor {
		return shared.FixPlanReasonNonConservativeUpgrade, true
	}
	return "", false
}

func formatFindingExample(finding shared.Finding) string {
	parts := []string{strings.TrimSpace(finding.ID)}
	if pkg := strings.TrimSpace(finding.PackageName); pkg != "" {
		parts = append(parts, fmt.Sprintf("package %s", pkg))
	} else if target := strings.TrimSpace(finding.Target); target != "" {
		parts = append(parts, fmt.Sprintf("target %s", target))
	}
	return strings.Join(parts, " — ")
}

type reasonAccumulator map[shared.FixPlanReasonCategory]*shared.FixPlanReason

func newReasonAccumulator() reasonAccumulator {
	return reasonAccumulator{
		shared.FixPlanReasonNoFixedVersion:         {Category: shared.FixPlanReasonNoFixedVersion},
		shared.FixPlanReasonManualChange:           {Category: shared.FixPlanReasonManualChange},
		shared.FixPlanReasonNonConservativeUpgrade: {Category: shared.FixPlanReasonNonConservativeUpgrade},
		shared.FixPlanReasonOutsideScope:           {Category: shared.FixPlanReasonOutsideScope},
	}
}

func (a reasonAccumulator) Add(category shared.FixPlanReasonCategory, example string) {
	item, ok := a[category]
	if !ok {
		return
	}
	item.Count++
	if example == "" || len(item.Examples) >= 3 {
		return
	}
	item.Examples = append(item.Examples, example)
}

func (a reasonAccumulator) List() []shared.FixPlanReason {
	ordered := []shared.FixPlanReasonCategory{
		shared.FixPlanReasonNoFixedVersion,
		shared.FixPlanReasonManualChange,
		shared.FixPlanReasonNonConservativeUpgrade,
		shared.FixPlanReasonOutsideScope,
	}
	result := make([]shared.FixPlanReason, 0, len(ordered))
	for _, category := range ordered {
		item := a[category]
		if item == nil || item.Count == 0 {
			continue
		}
		result = append(result, *item)
	}
	return result
}

func dedupeOperations(ops []shared.FixOperation) []shared.FixOperation {
	byPackage := map[string]shared.FixOperation{}
	for _, op := range ops {
		existing, ok := byPackage[op.PackageName]
		if !ok || shared.CompareVersions(existing.ProposedVersion, op.ProposedVersion) < 0 {
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
