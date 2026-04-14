package output

import (
	"strings"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func TestToSPDX_EmptyDeps(t *testing.T) {
	t.Parallel()
	data := ToSPDX(nil, "/root", "0.4.3")
	text := string(data)
	if !strings.Contains(text, "SPDXVersion: SPDX-2.3") {
		t.Error("expected SPDXVersion header")
	}
	if !strings.Contains(text, "DataLicense: CC0-1.0") {
		t.Error("expected DataLicense")
	}
}

func TestToSPDX_SingleDep(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"}}
	data := ToSPDX(deps, "/root", "0.4.3")
	text := string(data)
	if !strings.Contains(text, "PackageName: lodash") {
		t.Error("expected PackageName: lodash")
	}
	if !strings.Contains(text, "PackageVersion: 4.17.21") {
		t.Error("expected PackageVersion: 4.17.21")
	}
}

func TestToSPDX_MultipleDeps(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{
		{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"},
		{Name: "express", Version: "4.18.2", Ecosystem: "npm"},
	}
	data := ToSPDX(deps, "/root", "0.4.3")
	text := string(data)
	if !strings.Contains(text, "PackageName: lodash") {
		t.Error("expected lodash")
	}
	if !strings.Contains(text, "PackageName: express") {
		t.Error("expected express")
	}
	if strings.Count(text, "SPDXID: SPDXRef-Package-") != 2 {
		t.Errorf("expected 2 package IDs, got %d", strings.Count(text, "SPDXID: SPDXRef-Package-"))
	}
}

func TestToSPDX_PackagePurl(t *testing.T) {
	t.Parallel()
	deps := []shared.Dependency{{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"}}
	data := ToSPDX(deps, "/root", "0.4.3")
	text := string(data)
	if !strings.Contains(text, "pkg:npm/lodash@4.17.21") {
		t.Errorf("expected purl, got:\n%s", text)
	}
}

func TestToSPDX_DocumentMetadata(t *testing.T) {
	t.Parallel()
	data := ToSPDX(nil, "/root", "0.4.3")
	text := string(data)
	if !strings.Contains(text, "Creator: Tool: Celador-0.4.3") {
		t.Error("expected creator metadata")
	}
	if !strings.Contains(text, "Created:") {
		t.Error("expected created timestamp")
	}
}

func TestSanitizeSPDXID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"lodash", "lodash"},
		{"@scope/pkg", "-scope-pkg"},
		{"react-dom", "react-dom"},
		{"my.package", "my.package"},
		{"express", "express"},
	}
	for _, tt := range tests {
		result := sanitizeSPDXID(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeSPDXID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
