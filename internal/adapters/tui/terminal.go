package tui

import (
	"bufio"
	"context"
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

func (ui *TerminalUI) RenderScan(_ context.Context, result shared.ScanResult) error {
	renderedFindings := shared.RenderedFindingLines(result.Findings)

	_, err := fmt.Fprintf(ui.out, "Scan fingerprint: %s\nFindings: %d (ignored: %d)\n", result.Fingerprint, len(renderedFindings), result.IgnoredCount)
	if err != nil {
		return err
	}
	for _, line := range renderedFindings {
		if _, err := fmt.Fprintf(ui.out, "- %s\n", line); err != nil {
			return err
		}
	}
	if result.FromCache {
		_, err = fmt.Fprintln(ui.out, "Result source: cache")
	}
	if result.OfflineFallback {
		_, err = fmt.Fprintln(ui.out, "Mode: offline fallback")
	}
	return err
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
