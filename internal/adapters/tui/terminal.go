package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
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
	_, err := fmt.Fprintf(ui.out, "Scan fingerprint: %s\nFindings: %d (ignored: %d)\n", result.Fingerprint, len(result.Findings), result.IgnoredCount)
	if err != nil {
		return err
	}
	for _, finding := range result.Findings {
		if _, err := fmt.Fprintf(ui.out, "- [%s] %s\n", finding.Severity, formatFindingLine(finding)); err != nil {
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

func formatFindingLine(finding shared.Finding) string {
	parts := []string{finding.ID}

	context := formatFindingContext(finding)
	if context != "" {
		parts = append(parts, context)
	}

	summary := strings.TrimSpace(finding.Summary)
	if summary == "" {
		summary = fallbackFindingSummary(finding)
	}
	if summary != "" {
		parts = append(parts, summary)
	}

	if finding.Fixable && finding.FixVersion != "" {
		parts = append(parts, fmt.Sprintf("fixed in %s", finding.FixVersion))
	}

	return strings.Join(parts, ": ")
}

func formatFindingContext(finding shared.Finding) string {
	switch finding.Source {
	case shared.FindingSourceOSV:
		if finding.PackageName != "" {
			return fmt.Sprintf("package %s", finding.PackageName)
		}
		if finding.Target != "" {
			return fmt.Sprintf("target %s", finding.Target)
		}
	default:
		if len(finding.Locations) > 0 {
			location := finding.Locations[0]
			if location.Path != "" {
				if location.Line > 0 {
					return fmt.Sprintf("target %s:%d", location.Path, location.Line)
				}
				return fmt.Sprintf("target %s", location.Path)
			}
		}
		if finding.Target != "" {
			return fmt.Sprintf("target %s", finding.Target)
		}
	}

	return ""
}

func fallbackFindingSummary(finding shared.Finding) string {
	if finding.Source == shared.FindingSourceOSV {
		pkg := firstNonEmpty(finding.PackageName, finding.Target)
		if pkg != "" {
			return fmt.Sprintf("Vulnerability detected in %s", pkg)
		}
		return "Vulnerability detected"
	}

	if len(finding.Locations) > 0 && finding.Locations[0].Path != "" {
		return fmt.Sprintf("Review %s", filepath.Base(finding.Locations[0].Path))
	}
	if finding.Target != "" {
		return fmt.Sprintf("Review %s", filepath.Base(finding.Target))
	}

	return "Review this finding"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func (ui *TerminalUI) RenderFixPlan(_ context.Context, plan shared.FixPlan) error {
	_, err := fmt.Fprintf(ui.out, "%s\n%s", plan.Summary, plan.DryRunDiff)
	return err
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
