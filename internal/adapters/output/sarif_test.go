package output

import (
	"encoding/json"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestToSARIF_EmptyFindings(t *testing.T) {
	t.Parallel()
	data := ToSARIF(nil, "0.4.2")
	var report SARIFReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if report.Version != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %q", report.Version)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(report.Runs))
	}
	if len(report.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(report.Runs[0].Results))
	}
}

func TestToSARIF_SingleFinding(t *testing.T) {
	t.Parallel()
	findings := []shared.Finding{
		{ID: "GHSA-xxxx", PackageName: "lodash", Target: "lodash", Summary: "Prototype pollution", Severity: shared.SeverityHigh},
	}
	data := ToSARIF(findings, "0.4.2")
	var report SARIFReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(report.Runs[0].Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Runs[0].Results))
	}
	result := report.Runs[0].Results[0]
	if result.RuleID != "GHSA-xxxx" {
		t.Errorf("expected ruleId GHSA-xxxx, got %q", result.RuleID)
	}
	if result.Level != "error" {
		t.Errorf("expected level 'error' for high severity, got %q", result.Level)
	}
	if result.Message.Text != "Prototype pollution" {
		t.Errorf("expected message, got %q", result.Message.Text)
	}
}

func TestToSARIF_MultipleFindings(t *testing.T) {
	t.Parallel()
	findings := []shared.Finding{
		{ID: "GHSA-aaaa", PackageName: "express", Target: "express", Summary: "XSS vulnerability", Severity: shared.SeverityCritical},
		{ID: "GHSA-bbbb", PackageName: "axios", Target: "axios", Summary: "SSRF", Severity: shared.SeverityMedium},
		{ID: "GHSA-cccc", PackageName: "chalk", Target: "chalk", Summary: "ReDoS", Severity: shared.SeverityLow},
	}
	data := ToSARIF(findings, "0.4.2")
	var report SARIFReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(report.Runs[0].Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(report.Runs[0].Results))
	}
}

func TestToSARIF_SeverityMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		severity shared.Severity
		expected string
	}{
		{shared.SeverityCritical, "error"},
		{shared.SeverityHigh, "error"},
		{shared.SeverityMedium, "warning"},
		{shared.SeverityLow, "note"},
	}
	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			result := severityToSARIFLevel(tt.severity)
			if result != tt.expected {
				t.Errorf("severityToSARIFLevel(%v) = %q, want %q", tt.severity, result, tt.expected)
			}
		})
	}
}

func TestToSARIF_ValidJSON(t *testing.T) {
	t.Parallel()
	findings := []shared.Finding{
		{ID: "GHSA-test", PackageName: "pkg", Target: "pkg", Summary: "Test", Severity: shared.SeverityHigh},
	}
	data := ToSARIF(findings, "0.4.2")
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestToSARIF_DriverMetadata(t *testing.T) {
	t.Parallel()
	data := ToSARIF(nil, "1.2.3")
	var report SARIFReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	driver := report.Runs[0].Tool.Driver
	if driver.Name != "Celador" {
		t.Errorf("expected driver name 'Celador', got %q", driver.Name)
	}
	if driver.Version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got %q", driver.Version)
	}
	if driver.InformationURI == "" {
		t.Error("expected non-empty informationUri")
	}
}
