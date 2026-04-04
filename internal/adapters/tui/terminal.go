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

func (ui *TerminalUI) RenderBrandingHeader(_ context.Context, version string) error {
	for _, line := range shared.CeladorBranding.ASCIIArt {
		if _, err := fmt.Fprintln(ui.out, line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(ui.out); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(ui.out, shared.CeladorBranding.Slogan); err != nil {
		return err
	}
	if strings.TrimSpace(version) != "" {
		if _, err := fmt.Fprintf(ui.out, "Version: %s\n", version); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(ui.out)
	return err
}

func (ui *TerminalUI) RenderInit(_ context.Context, report shared.InitReport) error {
	return ui.renderChecklistReport(report.Title, report.Subtitle, report.Sections)
}

func (ui *TerminalUI) renderChecklistReport(title string, subtitle string, sections []shared.ChecklistSection) error {
	if strings.TrimSpace(title) != "" {
		if _, err := fmt.Fprintln(ui.out, title); err != nil {
			return err
		}
	}
	if strings.TrimSpace(subtitle) != "" {
		if _, err := fmt.Fprintln(ui.out, subtitle); err != nil {
			return err
		}
	}
	for index, section := range sections {
		if index > 0 {
			if _, err := fmt.Fprintln(ui.out); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(ui.out, section.Title); err != nil {
			return err
		}
		if strings.TrimSpace(section.Summary) != "" {
			if _, err := fmt.Fprintln(ui.out, section.Summary); err != nil {
				return err
			}
		}
		for _, item := range section.Items {
			line := fmt.Sprintf("✓ %s", item.Label)
			if strings.TrimSpace(item.Value) != "" {
				line += fmt.Sprintf(" = %s", item.Value)
			}
			if badge := formatChecklistStatus(item.Status); badge != "" {
				line += " " + badge
			}
			if _, err := fmt.Fprintln(ui.out, line); err != nil {
				return err
			}
			if err := ui.writeIndentedBlock(item.Detail, "  "); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ui *TerminalUI) writeIndentedBlock(text string, indent string) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	for _, line := range strings.Split(trimmed, "\n") {
		if _, err := fmt.Fprintf(ui.out, "%s%s\n", indent, line); err != nil {
			return err
		}
	}
	return nil
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

func formatChecklistStatus(status shared.ChecklistStatus) string {
	switch status {
	case shared.ChecklistStatusNew:
		return "NEW"
	case shared.ChecklistStatusUpdated:
		return "UPDATED"
	case shared.ChecklistStatusUnchanged:
		return "OK"
	case shared.ChecklistStatusInfo:
		return "INFO"
	default:
		return ""
	}
}

func (ui *TerminalUI) RenderFixPlan(_ context.Context, plan shared.FixPlan) error {
	return ui.renderChecklistReport(renderFixPlanTitle(plan), renderFixPlanSubtitle(plan), buildFixPlanSections(plan))
}

func renderFixPlanTitle(plan shared.FixPlan) string {
	if len(plan.Operations) == 0 {
		return "Remediation analysis complete"
	}
	return "Remediation plan ready"
}

func renderFixPlanSubtitle(plan shared.FixPlan) string {
	if len(plan.Operations) == 0 {
		return "No conservative package.json changes were queued automatically. Review the findings below before making manual changes."
	}
	return "Celador prepared conservative package.json changes and a diff preview for review before apply."
}

func buildFixPlanSections(plan shared.FixPlan) []shared.ChecklistSection {
	sections := []shared.ChecklistSection{buildFixPlanSummarySection(plan)}
	if len(plan.Operations) > 0 {
		sections = append(sections, buildFixPlanOperationsSection(plan.Operations))
	}
	if len(plan.Reasons) > 0 {
		sections = append(sections, buildFixPlanReasonsSection(plan))
	}
	sections = append(sections, buildFixPlanDiffSection(plan))
	return sections
}

func buildFixPlanSummarySection(plan shared.FixPlan) shared.ChecklistSection {
	reviewCount := 0
	for _, reason := range plan.Reasons {
		reviewCount += reason.Count
	}
	diffState := "ready"
	if strings.TrimSpace(plan.DryRunDiff) == "" || len(plan.Operations) == 0 {
		diffState = "no manifest changes"
	}
	return shared.ChecklistSection{
		Title:   "Plan summary",
		Summary: plan.Summary,
		Items: []shared.ChecklistItem{
			{Label: "safe operations", Value: fmt.Sprintf("%d", len(plan.Operations)), Status: shared.ChecklistStatusUnchanged},
			{Label: "findings left for review", Value: fmt.Sprintf("%d", reviewCount), Status: reviewStatus(reviewCount)},
			{Label: "diff preview", Value: diffState, Status: shared.ChecklistStatusInfo},
		},
	}
}

func buildFixPlanOperationsSection(operations []shared.FixOperation) shared.ChecklistSection {
	items := make([]shared.ChecklistItem, 0, len(operations))
	for _, op := range operations {
		items = append(items, shared.ChecklistItem{
			Label:  op.PackageName,
			Value:  formatOperationValue(op),
			Status: operationStatus(op),
			Detail: formatOperationDetail(op),
		})
	}
	return shared.ChecklistSection{
		Title:   "Planned operations",
		Summary: fmt.Sprintf("Prepared %d conservative manifest %s that can be applied automatically.", len(operations), pluralize(len(operations), "change", "changes")),
		Items:   items,
	}
}

func buildFixPlanReasonsSection(plan shared.FixPlan) shared.ChecklistSection {
	title := "Findings left for review"
	summary := "These findings were detected but not added to the automatic remediation plan."
	if len(plan.Operations) == 0 {
		title = "Why nothing was planned"
		summary = "Celador finished the analysis, but every finding still needs manual review or upstream fixes."
	}
	items := make([]shared.ChecklistItem, 0, len(plan.Reasons))
	for _, reason := range plan.Reasons {
		items = append(items, shared.ChecklistItem{
			Label:  formatFixReasonLabel(reason),
			Value:  fmt.Sprintf("%d %s", reason.Count, pluralize(reason.Count, "finding", "findings")),
			Status: shared.ChecklistStatusInfo,
			Detail: formatReasonExamples(reason.Examples),
		})
	}
	return shared.ChecklistSection{Title: title, Summary: summary, Items: items}
}

func buildFixPlanDiffSection(plan shared.FixPlan) shared.ChecklistSection {
	state := "ready"
	if len(plan.Operations) == 0 {
		state = "no changes"
	}
	return shared.ChecklistSection{
		Title:   "Diff preview",
		Summary: "Review the package.json patch preview before applying the plan.",
		Items: []shared.ChecklistItem{{
			Label:  "package.json diff",
			Value:  state,
			Status: shared.ChecklistStatusInfo,
			Detail: strings.TrimRight(plan.DryRunDiff, "\n"),
		}},
	}
}

func formatOperationValue(op shared.FixOperation) string {
	if op.Strategy == "override" {
		return fmt.Sprintf("override -> %s", op.ProposedVersion)
	}
	if op.ManifestSection != "" {
		return fmt.Sprintf("%s %s -> %s", op.ManifestSection, op.CurrentVersion, op.ProposedVersion)
	}
	return fmt.Sprintf("%s -> %s", op.CurrentVersion, op.ProposedVersion)
}

func formatOperationDetail(op shared.FixOperation) string {
	parts := []string{formatOperationLine(op)}
	if strings.TrimSpace(op.File) != "" {
		parts = append(parts, fmt.Sprintf("File: %s", op.File))
	}
	if op.RequiresInstall {
		parts = append(parts, "Follow-up: reinstall dependencies to refresh the lockfile.")
	}
	return strings.Join(parts, "\n")
}

func operationStatus(op shared.FixOperation) shared.ChecklistStatus {
	if op.Strategy == "override" {
		return shared.ChecklistStatusNew
	}
	return shared.ChecklistStatusUpdated
}

func reviewStatus(count int) shared.ChecklistStatus {
	if count == 0 {
		return shared.ChecklistStatusUnchanged
	}
	return shared.ChecklistStatusInfo
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
	label := formatFixReasonLabel(reason)
	switch reason.Category {
	case shared.FixPlanReasonNoFixedVersion:
		label = pluralize(reason.Count, "no known fixed version", "no known fixed version")
	case shared.FixPlanReasonManualChange:
		label = pluralize(reason.Count, "manual remediation required", "manual remediation required")
	case shared.FixPlanReasonNonConservativeUpgrade:
		label = pluralize(reason.Count, "non-conservative upgrade requires review", "non-conservative upgrade requires review")
	case shared.FixPlanReasonOutsideScope:
		label = pluralize(reason.Count, "outside current remediation scope", "outside current remediation scope")
	}
	if len(reason.Examples) == 0 {
		return label
	}
	return fmt.Sprintf("%s (%s)", label, strings.Join(reason.Examples, "; "))
}

func formatFixReasonLabel(reason shared.FixPlanReason) string {
	switch reason.Category {
	case shared.FixPlanReasonNoFixedVersion:
		return "no known fixed version"
	case shared.FixPlanReasonManualChange:
		return "manual remediation required"
	case shared.FixPlanReasonNonConservativeUpgrade:
		return "non-conservative upgrade requires review"
	case shared.FixPlanReasonOutsideScope:
		return "outside current remediation scope"
	default:
		return "requires review"
	}
}

func formatReasonExamples(examples []string) string {
	if len(examples) == 0 {
		return ""
	}
	return fmt.Sprintf("Examples: %s", strings.Join(examples, "; "))
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func (ui *TerminalUI) RenderInstallAssessment(_ context.Context, assessment shared.InstallAssessment) error {
	return ui.renderChecklistReport("Install preflight", renderInstallSubtitle(assessment), buildInstallSections(assessment))
}

func (ui *TerminalUI) RenderInstallTimeline(_ context.Context, timeline shared.InstallTimeline) error {
	return ui.renderChecklistReport("Install timeline", renderInstallTimelineSubtitle(timeline), buildInstallTimelineSections(timeline))
}

func renderInstallSubtitle(assessment shared.InstallAssessment) string {
	manager := string(assessment.Manager)
	if manager == "" {
		manager = string(shared.PackageManagerUnknown)
	}
	if assessment.ShouldPrompt {
		return fmt.Sprintf("Assessment complete for %s before %s install. Approval is recommended before continuing.", assessment.Package, manager)
	}
	return fmt.Sprintf("Assessment complete for %s before %s install. No interactive approval is required from the current risk model.", assessment.Package, manager)
}

func buildInstallSections(assessment shared.InstallAssessment) []shared.ChecklistSection {
	sections := []shared.ChecklistSection{buildInstallSummarySection(assessment), buildInstallRiskSection(assessment)}
	if len(assessment.SuggestedArgs) > 0 {
		sections = append(sections, shared.ChecklistSection{
			Title:   "Suggested command",
			Summary: "Use the safer command below if you want Celador's recommended install arguments.",
			Items: []shared.ChecklistItem{{
				Label:  "command",
				Value:  strings.Join(assessment.SuggestedArgs, " "),
				Status: shared.ChecklistStatusInfo,
			}},
		})
	}
	return sections
}

func buildInstallSummarySection(assessment shared.InstallAssessment) shared.ChecklistSection {
	registryStatus := "known"
	registryState := shared.ChecklistStatusUnchanged
	if assessment.Unknown {
		registryStatus = "unknown"
		registryState = shared.ChecklistStatusInfo
	}
	return shared.ChecklistSection{
		Title:   "Package summary",
		Summary: "Review the package, manager, and registry confidence before installation.",
		Items: []shared.ChecklistItem{
			{Label: "package", Value: assessment.Package, Status: shared.ChecklistStatusUnchanged},
			{Label: "package manager", Value: string(assessment.Manager), Status: shared.ChecklistStatusUnchanged},
			{Label: "risk", Value: formatSeverityHeading(assessment.Risk), Status: installRiskStatus(assessment)},
			{Label: "registry status", Value: registryStatus, Status: registryState},
		},
	}
}

func buildInstallRiskSection(assessment shared.InstallAssessment) shared.ChecklistSection {
	section := shared.ChecklistSection{
		Title:   "Risk review",
		Summary: installRiskSummary(assessment),
	}
	if len(assessment.Reasons) == 0 {
		section.Items = []shared.ChecklistItem{{
			Label:  "assessment",
			Value:  "no specific concerns reported",
			Status: shared.ChecklistStatusUnchanged,
		}}
		return section
	}
	items := make([]shared.ChecklistItem, 0, len(assessment.Reasons))
	for _, reason := range assessment.Reasons {
		items = append(items, shared.ChecklistItem{
			Label:  reason,
			Status: shared.ChecklistStatusInfo,
		})
	}
	section.Items = items
	return section
}

func renderInstallTimelineSubtitle(timeline shared.InstallTimeline) string {
	packageName := timeline.Assessment.Package
	if packageName == "" && len(timeline.RequestedArgs) > 0 {
		packageName = timeline.RequestedArgs[0]
	}
	manager := string(timeline.Assessment.Manager)
	if manager == "" {
		manager = string(shared.PackageManagerUnknown)
	}
	switch timeline.ExecutionState {
	case shared.InstallExecutionRunning:
		return fmt.Sprintf("Execution started for %s with %s. Celador is narrating the real install lifecycle as it happens.", packageName, manager)
	case shared.InstallExecutionSucceeded:
		return fmt.Sprintf("Execution finished for %s with %s. The timeline below reflects the completed install flow.", packageName, manager)
	case shared.InstallExecutionFailed:
		return fmt.Sprintf("Execution stopped for %s with %s. The timeline below reflects the last completed step and the package manager failure.", packageName, manager)
	default:
		return fmt.Sprintf("Execution summary for %s with %s.", packageName, manager)
	}
}

func buildInstallTimelineSections(timeline shared.InstallTimeline) []shared.ChecklistSection {
	sections := []shared.ChecklistSection{
		buildInstallRequestSection(timeline),
		buildInstallApprovalSection(timeline),
		buildInstallExecutionSection(timeline),
		buildInstallSecuritySummarySection(timeline),
	}
	sections = append(sections, buildInstallOutcomeSection(timeline))
	return sections
}

func buildInstallRequestSection(timeline shared.InstallTimeline) shared.ChecklistSection {
	items := []shared.ChecklistItem{{
		Label:  "requested packages",
		Value:  strings.Join(timeline.RequestedArgs, " "),
		Status: shared.ChecklistStatusUnchanged,
	}}
	if len(timeline.Assessment.SuggestedArgs) > 0 {
		items = append(items, shared.ChecklistItem{
			Label:  "safer suggested command",
			Value:  strings.Join(timeline.Assessment.SuggestedArgs, " "),
			Status: shared.ChecklistStatusInfo,
		})
	}
	return shared.ChecklistSection{
		Title:   "1. Request and preflight",
		Summary: fmt.Sprintf("Preflight completed for %s with %s risk before package manager execution.", timeline.Assessment.Package, formatSeverityHeading(timeline.Assessment.Risk)),
		Items:   items,
	}
}

func buildInstallApprovalSection(timeline shared.InstallTimeline) shared.ChecklistSection {
	item := shared.ChecklistItem{Label: "approval status", Status: shared.ChecklistStatusUnchanged}
	summary := "Celador did not require an interactive approval for this install request."
	switch timeline.Approval {
	case shared.InstallApprovalPendingInteractive:
		item.Value = "review required"
		item.Status = shared.ChecklistStatusInfo
		summary = "Celador flagged the request for review before package manager execution."
	case shared.InstallApprovalAutoApproved:
		item.Value = "granted with --yes"
		item.Status = shared.ChecklistStatusInfo
		summary = "Approval was required by policy and was granted non-interactively with --yes."
	case shared.InstallApprovalPromptApproved:
		item.Value = "granted interactively"
		item.Status = shared.ChecklistStatusInfo
		summary = "Approval was required by policy and was granted interactively before execution."
	default:
		item.Value = "not required"
	}
	return shared.ChecklistSection{Title: "2. Approval decision", Summary: summary, Items: []shared.ChecklistItem{item}}
}

func buildInstallExecutionSection(timeline shared.InstallTimeline) shared.ChecklistSection {
	command := strings.Join(timeline.Command, " ")
	status := shared.ChecklistStatusInfo
	state := "starting"
	detail := "Celador is handing the request to the workspace package manager."
	summary := "Package manager execution has started."
	switch timeline.ExecutionState {
	case shared.InstallExecutionSucceeded:
		status = shared.ChecklistStatusUpdated
		state = "completed"
		detail = "The package manager finished without reporting an execution error to Celador."
		summary = "Package manager execution completed successfully."
	case shared.InstallExecutionFailed:
		state = "failed"
		detail = strings.TrimSpace(timeline.Failure)
		summary = "Package manager execution failed and Celador stopped the flow."
	case shared.InstallExecutionRunning:
		state = "running"
	}
	return shared.ChecklistSection{
		Title:   "3. Package manager execution",
		Summary: summary,
		Items: []shared.ChecklistItem{{
			Label:  "command",
			Value:  command,
			Status: status,
			Detail: fmt.Sprintf("State: %s\n%s", state, detail),
		}},
	}
}

func buildInstallSecuritySummarySection(timeline shared.InstallTimeline) shared.ChecklistSection {
	item := shared.ChecklistItem{
		Label:  "post-install security check",
		Value:  "not run automatically",
		Status: shared.ChecklistStatusInfo,
		Detail: "Celador did not run a dependency scan after installation in this command flow. Run `celador scan` to audit the updated lockfiles explicitly.",
	}
	if timeline.PostInstallScan {
		item.Value = "completed"
		item.Status = shared.ChecklistStatusUpdated
		item.Detail = "Celador ran a post-install dependency review after package manager execution."
	}
	return shared.ChecklistSection{
		Title:   "4. Security summary",
		Summary: "Post-install security reporting reflects only actions Celador actually performed.",
		Items:   []shared.ChecklistItem{item},
	}
}

func buildInstallOutcomeSection(timeline shared.InstallTimeline) shared.ChecklistSection {
	status := shared.ChecklistStatusInfo
	value := "in progress"
	summary := "Celador is waiting for the package manager to finish."
	switch timeline.ExecutionState {
	case shared.InstallExecutionSucceeded:
		status = shared.ChecklistStatusUpdated
		value = "install finished"
		summary = installOutcomeSummary(timeline)
	case shared.InstallExecutionFailed:
		value = "install failed"
		summary = installOutcomeSummary(timeline)
	}
	return shared.ChecklistSection{
		Title:   "5. Final outcome",
		Summary: summary,
		Items: []shared.ChecklistItem{{
			Label:  "result",
			Value:  value,
			Status: status,
		}},
	}
}

func installOutcomeSummary(timeline shared.InstallTimeline) string {
	approval := "no extra approval was required"
	switch timeline.Approval {
	case shared.InstallApprovalAutoApproved:
		approval = "approval was required and granted with --yes"
	case shared.InstallApprovalPromptApproved:
		approval = "approval was required and granted interactively"
	case shared.InstallApprovalPendingInteractive:
		approval = "approval is still pending"
	}
	if timeline.ExecutionState == shared.InstallExecutionFailed {
		return fmt.Sprintf("The install did not finish because the package manager returned an error; %s before execution.", approval)
	}
	return fmt.Sprintf("The install finished and %s before execution.", approval)
}

func installRiskStatus(assessment shared.InstallAssessment) shared.ChecklistStatus {
	if assessment.ShouldPrompt || assessment.Unknown || assessment.Risk == shared.SeverityHigh || assessment.Risk == shared.SeverityCritical {
		return shared.ChecklistStatusInfo
	}
	return shared.ChecklistStatusUnchanged
}

func installRiskSummary(assessment shared.InstallAssessment) string {
	if assessment.ShouldPrompt {
		return "Celador recommends an explicit approval before package manager execution."
	}
	if assessment.Unknown {
		return "Registry confidence is incomplete, but the current policy does not require an interactive approval."
	}
	return "The current assessment does not require additional approval before installation."
}

func (ui *TerminalUI) Printf(format string, args ...any) { _, _ = fmt.Fprintf(ui.out, format, args...) }
