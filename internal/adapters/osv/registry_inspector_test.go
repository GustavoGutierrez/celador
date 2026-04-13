package osv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestInspectPackage_DenoV1Unsupported(t *testing.T) {
	t.Parallel()
	inspector := NewRegistryInspector()
	assessment, err := inspector.InspectPackage(context.Background(), shared.PackageManagerDeno, "lodash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assessment.Package != "lodash" {
		t.Errorf("expected package 'lodash', got %q", assessment.Package)
	}
	if assessment.Manager != shared.PackageManagerDeno {
		t.Errorf("expected manager deno, got %v", assessment.Manager)
	}
	if !assessment.Unknown {
		t.Error("expected Unknown=true for Deno packages")
	}
	if !assessment.ShouldPrompt {
		t.Error("expected ShouldPrompt=true for Deno packages")
	}
	if len(assessment.Reasons) == 0 || !strings.Contains(assessment.Reasons[0], "deno") {
		t.Errorf("expected deno-related reason, got: %v", assessment.Reasons)
	}
}

func TestInspectPackage_NetworkFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	inspector := NewRegistryInspector()
	assessment, err := inspector.InspectPackage(context.Background(), shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assessment.Risk != shared.SeverityMedium {
		t.Errorf("expected medium risk on network failure, got %v", assessment.Risk)
	}
	if !assessment.Unknown || !assessment.ShouldPrompt {
		t.Error("expected Unknown=true and ShouldPrompt=true on network failure")
	}
}

func TestInspectPackage_PackageNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	inspector := NewRegistryInspector()
	assessment, err := inspector.InspectPackage(context.Background(), shared.PackageManagerNPM, "nonexistent-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assessment.Risk != shared.SeverityMedium {
		t.Errorf("expected medium risk on 404, got %v", assessment.Risk)
	}
	if !assessment.Unknown {
		t.Error("expected Unknown=true when package not found")
	}
}

func TestIsHexLike_ShortString(t *testing.T) {
	t.Parallel()
	// 8 chars, all hex - isHexLike returns true for any length all-hex string
	// The inspector adds the length check (>80) separately
	if !isHexLike("abc123de") {
		t.Error("expected all-hex string to return true")
	}
	// Non-hex chars should return false
	if isHexLike("not-hex") {
		t.Error("expected non-hex string to return false")
	}
}

func TestIsHexLike_LongHexString(t *testing.T) {
	t.Parallel()
	longHex := strings.Repeat("abcdef0123456789", 6) // 96 chars, all hex
	if !isHexLike(longHex) {
		t.Error("expected long hex string to return true")
	}
}

func TestIsHexLike_MixedContent(t *testing.T) {
	t.Parallel()
	if isHexLike("abcdefg123456789") {
		t.Error("expected mixed content (contains 'g') to return false")
	}
}

func TestMaxSeverity_ReturnsHigher(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		a, b     shared.Severity
		expected shared.Severity
	}{
		{"low vs medium", shared.SeverityLow, shared.SeverityMedium, shared.SeverityMedium},
		{"medium vs high", shared.SeverityMedium, shared.SeverityHigh, shared.SeverityHigh},
		{"high vs critical", shared.SeverityHigh, shared.SeverityCritical, shared.SeverityCritical},
		{"same severity", shared.SeverityMedium, shared.SeverityMedium, shared.SeverityMedium},
		{"reversed order", shared.SeverityHigh, shared.SeverityLow, shared.SeverityHigh},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxSeverity(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("maxSeverity(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
