package shared

import (
	"fmt"
	"path/filepath"
	"strings"
)

type RenderedFindingGroup struct {
	Severity Severity
	Lines    []string
}

var renderedSeverityOrder = []Severity{
	SeverityCritical,
	SeverityHigh,
	SeverityMedium,
	SeverityLow,
}

func RenderedFindingLines(findings []Finding) []string {
	groups := RenderedFindingGroups(findings)
	lines := make([]string, 0, len(findings))
	for _, group := range groups {
		lines = append(lines, group.Lines...)
	}
	return lines
}

func RenderedFindingGroups(findings []Finding) []RenderedFindingGroup {
	type findingGroup struct {
		severity Severity
		base     string
		items    []Finding
	}

	groups := make(map[string]*findingGroup, len(findings))
	order := make([]string, 0, len(findings))

	for _, finding := range findings {
		base := formatFindingLine(finding)
		key := renderedFindingKey(finding.Severity, base)
		group, ok := groups[key]
		if !ok {
			group = &findingGroup{severity: finding.Severity, base: base}
			groups[key] = group
			order = append(order, key)
		}
		group.items = append(group.items, finding)
	}

	groupedLines := make(map[Severity][]string, len(renderedSeverityOrder))
	seenSeverity := make(map[Severity]struct{}, len(renderedSeverityOrder))
	severityOrder := make([]Severity, 0, len(renderedSeverityOrder))
	for _, key := range order {
		group := groups[key]
		if _, ok := seenSeverity[group.severity]; !ok {
			seenSeverity[group.severity] = struct{}{}
			severityOrder = append(severityOrder, group.severity)
		}
		if len(group.items) == 1 {
			groupedLines[group.severity] = append(groupedLines[group.severity], formatRenderedFinding(group.severity, group.base))
			continue
		}

		detailed := uniqueDetailedFindingLines(group.severity, group.items)
		if len(detailed) > 1 {
			groupedLines[group.severity] = append(groupedLines[group.severity], detailed...)
			continue
		}

		groupedLines[group.severity] = append(groupedLines[group.severity], formatRenderedFinding(group.severity, group.base))
	}

	renderedGroups := make([]RenderedFindingGroup, 0, len(groupedLines))
	added := make(map[Severity]struct{}, len(groupedLines))
	for _, severity := range renderedSeverityOrder {
		lines := groupedLines[severity]
		if len(lines) == 0 {
			continue
		}
		added[severity] = struct{}{}
		renderedGroups = append(renderedGroups, RenderedFindingGroup{Severity: severity, Lines: lines})
	}
	for _, severity := range severityOrder {
		if _, ok := added[severity]; ok {
			continue
		}
		lines := groupedLines[severity]
		if len(lines) == 0 {
			continue
		}
		renderedGroups = append(renderedGroups, RenderedFindingGroup{Severity: severity, Lines: lines})
	}

	return renderedGroups
}

func RenderedFindingCount(findings []Finding) int {
	return len(RenderedFindingLines(findings))
}

func RenderedFindings(findings []Finding) []Finding {
	type findingGroup struct {
		severity Severity
		base     string
		items    []Finding
	}

	groups := make(map[string]*findingGroup, len(findings))
	order := make([]string, 0, len(findings))

	for _, finding := range findings {
		base := formatFindingLine(finding)
		key := renderedFindingKey(finding.Severity, base)
		group, ok := groups[key]
		if !ok {
			group = &findingGroup{severity: finding.Severity, base: base}
			groups[key] = group
			order = append(order, key)
		}
		group.items = append(group.items, finding)
	}

	rendered := make([]Finding, 0, len(order))
	for _, key := range order {
		group := groups[key]
		if len(group.items) == 1 {
			rendered = append(rendered, group.items[0])
			continue
		}

		detailed := map[string][]Finding{}
		detailedOrder := make([]string, 0, len(group.items))
		for _, finding := range group.items {
			line := formatRenderedFinding(group.severity, formatFindingLineWithContext(finding, formatDuplicateFindingContext(finding)))
			if _, ok := detailed[line]; !ok {
				detailedOrder = append(detailedOrder, line)
			}
			detailed[line] = append(detailed[line], finding)
		}

		if len(detailedOrder) > 1 {
			for _, line := range detailedOrder {
				rendered = append(rendered, pickRenderedFindingRepresentative(detailed[line]))
			}
			continue
		}

		rendered = append(rendered, pickRenderedFindingRepresentative(group.items))
	}

	return rendered
}

func uniqueDetailedFindingLines(severity Severity, findings []Finding) []string {
	seen := make(map[string]struct{}, len(findings))
	lines := make([]string, 0, len(findings))

	for _, finding := range findings {
		line := formatRenderedFinding(severity, formatFindingLineWithContext(finding, formatDuplicateFindingContext(finding)))
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		lines = append(lines, line)
	}

	return lines
}

func pickRenderedFindingRepresentative(findings []Finding) Finding {
	best := findings[0]
	for _, finding := range findings[1:] {
		if findingRank(finding) > findingRank(best) {
			best = finding
			continue
		}
		if findingRank(finding) == findingRank(best) && CompareVersions(best.FixVersion, finding.FixVersion) < 0 {
			best = finding
		}
	}
	return best
}

func findingRank(finding Finding) int {
	rank := 0
	if finding.Source == FindingSourceOSV {
		rank += 4
	}
	if finding.PackageName != "" {
		rank += 2
	}
	if finding.Fixable && finding.FixVersion != "" {
		rank += 1
	}
	return rank
}

func renderedFindingKey(severity Severity, line string) string {
	return string(severity) + "\x00" + line
}

func formatRenderedFinding(severity Severity, line string) string {
	return fmt.Sprintf("[%s] %s", severity, line)
}

func formatFindingLine(finding Finding) string {
	return formatFindingLineWithContext(finding, formatFindingContext(finding))
}

func formatFindingLineWithContext(finding Finding, context string) string {
	parts := []string{finding.ID}

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

func formatDuplicateFindingContext(finding Finding) string {
	context := formatFindingContext(finding)
	extra := duplicateFindingExtraContext(finding)
	if extra == "" {
		return context
	}
	if context == "" {
		return extra
	}
	return fmt.Sprintf("%s (%s)", context, extra)
}

func duplicateFindingExtraContext(finding Finding) string {
	if finding.Source != FindingSourceOSV {
		return ""
	}

	pkg := strings.TrimSpace(finding.PackageName)
	target := strings.TrimSpace(finding.Target)
	if pkg == "" || target == "" || pkg == target {
		return ""
	}

	return fmt.Sprintf("target %s", target)
}

func formatFindingContext(finding Finding) string {
	switch finding.Source {
	case FindingSourceOSV:
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

func fallbackFindingSummary(finding Finding) string {
	if finding.Source == FindingSourceOSV {
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
