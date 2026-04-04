package shared

import "testing"

func TestRenderedFindingGroupsOrdersBySeverity(t *testing.T) {
	t.Parallel()

	groups := RenderedFindingGroups([]Finding{
		{ID: "rule-medium", Source: FindingSourceRule, Severity: SeverityMedium, Target: "src/app.tsx", Summary: "Review dynamic class usage"},
		{ID: "GHSA-critical", Source: FindingSourceOSV, Severity: SeverityCritical, PackageName: "axios", Target: "axios", Summary: "Remote code execution in redirect handling"},
		{ID: "GHSA-high", Source: FindingSourceOSV, Severity: SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "Prototype pollution in merge helper", FixVersion: "4.17.21", Fixable: true},
	})

	if len(groups) != 3 {
		t.Fatalf("expected three severity groups, got %d", len(groups))
	}
	if groups[0].Severity != SeverityCritical || groups[1].Severity != SeverityHigh || groups[2].Severity != SeverityMedium {
		t.Fatalf("unexpected severity order: %+v", groups)
	}
	if len(groups[1].Lines) != 1 || groups[1].Lines[0] != "[high] GHSA-high: package lodash: Prototype pollution in merge helper: fixed in 4.17.21" {
		t.Fatalf("expected high severity group line with fixed version, got %+v", groups[1].Lines)
	}
	if RenderedFindingCount([]Finding{
		{ID: "GHSA-high", Source: FindingSourceOSV, Severity: SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "Prototype pollution in merge helper", FixVersion: "4.17.21", Fixable: true},
		{ID: "GHSA-high", Source: FindingSourceOSV, Severity: SeverityHigh, PackageName: "lodash", Target: "lodash", Summary: "Prototype pollution in merge helper", FixVersion: "4.17.21", Fixable: true},
	}) != 1 {
		t.Fatalf("expected deduplicated rendered finding count")
	}
}
