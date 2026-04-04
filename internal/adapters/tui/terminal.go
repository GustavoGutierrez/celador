package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"golang.org/x/term"
)

type TerminalUI struct {
	in  io.Reader
	out io.Writer
	tty bool
	ci  bool
}

func NewTerminalUI(in io.Reader, out io.Writer, tty bool, ci bool) *TerminalUI {
	return &TerminalUI{in: in, out: out, tty: tty, ci: ci}
}

func IsTTY(_ uintptr, _ uintptr) bool {
	return term.IsTerminal(0) && term.IsTerminal(1)
}

func (ui *TerminalUI) Confirm(_ context.Context, prompt string) (bool, error) {
	if !ui.tty || ui.ci {
		return false, fmt.Errorf("prompt unavailable in non-interactive mode")
	}
	if _, err := fmt.Fprintf(ui.out, "%s [y/N]: ", prompt); err != nil {
		return false, err
	}
	line, err := bufio.NewReader(ui.in).ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes", nil
}

func (ui *TerminalUI) RenderScan(_ context.Context, result shared.ScanResult, options shared.ScanRenderOptions) error {
	if options.Format == shared.ScanRenderFormatJSON {
		return ui.renderScanJSON(result)
	}
	return ui.renderScanText(result, options)
}

func (ui *TerminalUI) renderScanText(result shared.ScanResult, options shared.ScanRenderOptions) error {
	renderedCount := shared.RenderedFindingCount(result.Findings)
	if _, err := fmt.Fprintf(ui.out, "Scan fingerprint: %s\nFindings: %d (ignored: %d)\n", result.Fingerprint, renderedCount, result.IgnoredCount); err != nil {
		return err
	}
	if options.Verbose {
		if _, err := fmt.Fprintf(ui.out, "Dependencies scanned: %d\nPackage manager: %s\n", len(result.Dependencies), result.Workspace.PackageManager); err != nil {
			return err
		}
		if result.RuleVersion != "" {
			if _, err := fmt.Fprintf(ui.out, "Rule pack: %s\n", result.RuleVersion); err != nil {
				return err
			}
		}
	}

	groups := shared.RenderedFindingGroups(result.Findings)
	if len(groups) == 0 {
		if _, err := fmt.Fprintln(ui.out, "Status: no actionable findings"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(ui.out); err != nil {
			return err
		}
		for index, group := range groups {
			if _, err := fmt.Fprintf(ui.out, "%s findings:\n", formatSeverityHeading(group.Severity)); err != nil {
				return err
			}
			for _, line := range group.Lines {
				if _, err := fmt.Fprintf(ui.out, "- %s\n", line); err != nil {
					return err
				}
			}
			if options.Verbose && index < len(groups)-1 {
				if _, err := fmt.Fprintln(ui.out); err != nil {
					return err
				}
			}
		}
	}

	if result.FromCache {
		if _, err := fmt.Fprintln(ui.out, "Result source: cache"); err != nil {
			return err
		}
	}
	if result.OfflineFallback {
		if _, err := fmt.Fprintln(ui.out, "Mode: offline fallback"); err != nil {
			return err
		}
	}
	return nil
}

func (ui *TerminalUI) renderScanJSON(result shared.ScanResult) error {
	type jsonCache struct {
		FromCache       bool `json:"from_cache"`
		OfflineFallback bool `json:"offline_fallback"`
	}
	type jsonWorkspace struct {
		Root           string                `json:"root,omitempty"`
		PackageManager shared.PackageManager `json:"package_manager"`
		Lockfiles      []string              `json:"lockfiles,omitempty"`
	}
	type jsonFinding struct {
		ID          string                   `json:"id"`
		Source      shared.FindingSource     `json:"source"`
		Severity    shared.Severity          `json:"severity"`
		PackageName string                   `json:"package_name,omitempty"`
		Target      string                   `json:"target,omitempty"`
		Summary     string                   `json:"summary"`
		Fixable     bool                     `json:"fixable"`
		FixVersion  string                   `json:"fix_version,omitempty"`
		Locations   []shared.FindingLocation `json:"locations,omitempty"`
		Rendered    string                   `json:"rendered"`
	}
	payload := struct {
		Fingerprint          string        `json:"fingerprint"`
		RenderedFindingCount int           `json:"rendered_finding_count"`
		RawFindingCount      int           `json:"raw_finding_count"`
		IgnoredCount         int           `json:"ignored_count"`
		DependencyCount      int           `json:"dependency_count"`
		RuleVersion          string        `json:"rule_version,omitempty"`
		GeneratedAt          string        `json:"generated_at,omitempty"`
		Cache                jsonCache     `json:"cache"`
		Workspace            jsonWorkspace `json:"workspace"`
		Findings             []jsonFinding `json:"findings"`
	}{
		Fingerprint:          result.Fingerprint,
		RenderedFindingCount: shared.RenderedFindingCount(result.Findings),
		RawFindingCount:      len(result.Findings),
		IgnoredCount:         result.IgnoredCount,
		DependencyCount:      len(result.Dependencies),
		RuleVersion:          result.RuleVersion,
		Cache: jsonCache{
			FromCache:       result.FromCache,
			OfflineFallback: result.OfflineFallback,
		},
		Workspace: jsonWorkspace{
			Root:           result.Workspace.Root,
			PackageManager: result.Workspace.PackageManager,
			Lockfiles:      result.Workspace.Lockfiles,
		},
		Findings: make([]jsonFinding, 0, len(result.Findings)),
	}
	if !result.GeneratedAt.IsZero() {
		payload.GeneratedAt = result.GeneratedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	for _, finding := range result.Findings {
		payload.Findings = append(payload.Findings, jsonFinding{
			ID:          finding.ID,
			Source:      finding.Source,
			Severity:    finding.Severity,
			PackageName: finding.PackageName,
			Target:      finding.Target,
			Summary:     renderedFindingSummary(finding),
			Fixable:     finding.Fixable,
			FixVersion:  finding.FixVersion,
			Locations:   finding.Locations,
			Rendered:    renderFindingLine(finding),
		})
	}

	encoder := json.NewEncoder(ui.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func renderedFindingSummary(finding shared.Finding) string {
	summary := strings.TrimSpace(finding.Summary)
	if summary != "" {
		return summary
	}
	return strings.TrimSpace(renderFindingLine(finding))
}

func renderFindingLine(finding shared.Finding) string {
	lines := shared.RenderedFindingLines([]shared.Finding{finding})
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

func formatSeverityHeading(severity shared.Severity) string {
	text := string(severity)
	if text == "" {
		return "Unspecified"
	}
	return strings.ToUpper(text[:1]) + text[1:]
}

func (ui *TerminalUI) RenderFixPlan(_ context.Context, plan shared.FixPlan) error {
	if _, err := fmt.Fprintf(ui.out, "%s\n", plan.Summary); err != nil {
		return err
	}
	if len(plan.Operations) > 0 {
		if _, err := fmt.Fprintln(ui.out, "Planned operations:"); err != nil {
			return err
		}
		for _, op := range plan.Operations {
			if _, err := fmt.Fprintf(ui.out, "- %s\n", formatOperationLine(op)); err != nil {
				return err
			}
		}
	}
	if len(plan.Reasons) > 0 {
		heading := "Unplanned findings:"
		if len(plan.Operations) == 0 {
			heading = "Why no safe remediation was planned:"
		}
		if _, err := fmt.Fprintln(ui.out, heading); err != nil {
			return err
		}
		for _, reason := range plan.Reasons {
			if _, err := fmt.Fprintf(ui.out, "- %s\n", formatFixPlanReason(reason)); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintf(ui.out, "%s", plan.DryRunDiff)
	return err
}

func formatOperationLine(op shared.FixOperation) string {
	if op.Strategy == "override" {
		return fmt.Sprintf("override %s to %s in package.json", op.PackageName, op.ProposedVersion)
	}
	if op.ManifestSection != "" {
		return fmt.Sprintf("bump %s in %s from %s to %s", op.PackageName, op.ManifestSection, op.CurrentVersion, op.ProposedVersion)
	}
	return fmt.Sprintf("bump %s from %s to %s", op.PackageName, op.CurrentVersion, op.ProposedVersion)
}

func formatFixPlanReason(reason shared.FixPlanReason) string {
	label := "finding requires review"
	switch reason.Category {
	case shared.FixPlanReasonNoFixedVersion:
		label = pluralize(reason.Count, "finding has no known fixed version", "findings have no known fixed version")
	case shared.FixPlanReasonManualChange:
		label = pluralize(reason.Count, "finding requires manual changes", "findings require manual changes")
	case shared.FixPlanReasonOutsideScope:
		label = pluralize(reason.Count, "finding is outside the current remediation scope", "findings are outside the current remediation scope")
	}
	if len(reason.Examples) == 0 {
		return label
	}
	return fmt.Sprintf("%s (%s)", label, strings.Join(reason.Examples, "; "))
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func (ui *TerminalUI) RenderInstallAssessment(_ context.Context, assessment shared.InstallAssessment) error {
	_, err := fmt.Fprintf(ui.out, "Preflight %s risk=%s unknown=%t\n", assessment.Package, assessment.Risk, assessment.Unknown)
	if err != nil {
		return err
	}
	for _, reason := range assessment.Reasons {
		if _, err := fmt.Fprintf(ui.out, "- %s\n", reason); err != nil {
			return err
		}
	}
	return nil
}

func (ui *TerminalUI) Printf(format string, args ...any) { _, _ = fmt.Fprintf(ui.out, format, args...) }
