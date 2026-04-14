package audit

import (
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestDetectTyposquat_ExactMatch_NoAlert(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{
		{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"},
		{Name: "react", Version: "18.2.0", Ecosystem: "npm"},
		{Name: "express", Version: "4.18.2", Ecosystem: "npm"},
	}
	findings := DetectTyposquat(deps)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for exact matches, got %d", len(findings))
	}
}

func TestDetectTyposquat_SingleCharSwap(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{
		{Name: "lodahs", Version: "4.17.21", Ecosystem: "npm"},
	}
	findings := DetectTyposquat(deps)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].SuspectedName != "lodahs" {
		t.Errorf("expected suspected 'lodahs', got %q", findings[0].SuspectedName)
	}
	if findings[0].LikelyTarget != "lodash" {
		t.Errorf("expected target 'lodash', got %q", findings[0].LikelyTarget)
	}
	// lodahs -> lodash is distance 2 (two chars swapped)
	if findings[0].Distance != 2 {
		t.Errorf("expected distance 2, got %d", findings[0].Distance)
	}
	if findings[0].Severity != shared.SeverityMedium {
		t.Errorf("expected severity medium for distance 2, got %v", findings[0].Severity)
	}
}

func TestDetectTyposquat_MissingChar(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{
		{Name: "reacts", Version: "18.2.0", Ecosystem: "npm"},
	}
	findings := DetectTyposquat(deps)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].LikelyTarget != "react" {
		t.Errorf("expected target 'react', got %q", findings[0].LikelyTarget)
	}
	if findings[0].Distance != 1 {
		t.Errorf("expected distance 1, got %d", findings[0].Distance)
	}
}

func TestDetectTyposquat_ExtraChar(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{
		{Name: "reacct", Version: "18.2.0", Ecosystem: "npm"},
	}
	findings := DetectTyposquat(deps)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].LikelyTarget != "react" {
		t.Errorf("expected target 'react', got %q", findings[0].LikelyTarget)
	}
}

func TestDetectTyposquat_DifferentPackage(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{
		{Name: "my-internal-utils", Version: "1.0.0", Ecosystem: "npm"},
	}
	findings := DetectTyposquat(deps)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for unrelated package name, got %d", len(findings))
	}
}

func TestDetectTyposquat_EmptyDeps(t *testing.T) {
	t.Parallel()
	findings := DetectTyposquat(nil)
	if findings != nil {
		t.Errorf("expected nil findings for empty deps, got %v", findings)
	}
}

func TestDetectTyposquat_Distance2_MediumSeverity(t *testing.T) {
	t.Parallel()
	// "axois" is distance 2 from "axios" (swap o and i)
	deps := []shared.Dependency{
		{Name: "axois", Version: "1.0.0", Ecosystem: "npm"},
	}
	findings := DetectTyposquat(deps)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Distance != 2 {
		t.Errorf("expected distance 2, got %d", findings[0].Distance)
	}
	if findings[0].Severity != shared.SeverityMedium {
		t.Errorf("expected severity medium for distance 2, got %v", findings[0].Severity)
	}
}

func TestDetectTyposquat_MultipleFindings(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{
		{Name: "lodahs", Version: "1.0.0", Ecosystem: "npm"},
		{Name: "reacts", Version: "1.0.0", Ecosystem: "npm"},
		{Name: "legit-package", Version: "1.0.0", Ecosystem: "npm"},
	}
	findings := DetectTyposquat(deps)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
}

func TestLevenshteinDistance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "abcd", 1},
		{"kitten", "sitting", 3},
		{"sunday", "saturday", 3},
		{"lodash", "lodahs", 2},
		{"react", "reacts", 1},
		{"axios", "axioss", 1},
		{"cross-env", "crossenv", 1},
		{"color-name", "colour-name", 1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := levenshtein(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestTopNpmPackages_Loaded(t *testing.T) {
	t.Parallel()
	if len(topNpmPackages) < 10 {
		t.Errorf("expected at least 10 known packages, got %d", len(topNpmPackages))
	}
}
