package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type PatchWriter struct{ fs ports.FileSystem }

func NewPatchWriter(fs ports.FileSystem) *PatchWriter { return &PatchWriter{fs: fs} }

func (w *PatchWriter) Preview(_ context.Context, _ shared.Workspace, plan shared.FixPlan) (string, error) {
	return plan.DryRunDiff, nil
}

func (w *PatchWriter) Apply(ctx context.Context, workspace shared.Workspace, plan shared.FixPlan) error {
	if workspace.ManifestPath == "" {
		return fmt.Errorf("workspace has no package.json")
	}
	body, err := w.fs.ReadFile(ctx, workspace.ManifestPath)
	if err != nil {
		return err
	}
	var pkg map[string]any
	if err := json.Unmarshal(body, &pkg); err != nil {
		return err
	}
	deps := ensureMap(pkg, "dependencies")
	overrides := ensureMap(pkg, "overrides")
	for _, op := range plan.Operations {
		if _, ok := deps[op.PackageName]; ok && op.Strategy == "bump" {
			deps[op.PackageName] = op.ProposedVersion
			continue
		}
		overrides[op.PackageName] = op.ProposedVersion
	}
	pkg["dependencies"] = deps
	if len(overrides) > 0 {
		pkg["overrides"] = overrides
	}
	formatted, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}
	formatted = append(formatted, '\n')
	return w.fs.WriteFile(ctx, workspace.ManifestPath, formatted)
}

func ensureMap(pkg map[string]any, key string) map[string]any {
	if existing, ok := pkg[key].(map[string]any); ok {
		return existing
	}
	result := map[string]any{}
	pkg[key] = result
	return result
}

func RenderPlanDiff(ops []shared.FixOperation) string {
	if len(ops) == 0 {
		return "No diff available.\n"
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].PackageName < ops[j].PackageName })
	var builder strings.Builder
	for _, op := range ops {
		builder.WriteString(fmt.Sprintf("--- %s\n+++ %s\n", op.File, op.File))
		builder.WriteString(fmt.Sprintf("- %s@%s\n+ %s@%s\n", op.PackageName, op.CurrentVersion, op.PackageName, op.ProposedVersion))
	}
	return builder.String()
}
