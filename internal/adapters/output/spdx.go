package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

// ToSPDX generates an SPDX 2.3 tag-value SBOM from the dependency list.
func ToSPDX(deps []shared.Dependency, wsRoot string, toolVersion string) []byte {
	var b strings.Builder
	ns := fmt.Sprintf("https://spdx.org/spdxdocs/celador-%s-%s", wsRoot, time.Now().UTC().Format("20060102T150405Z"))

	b.WriteString("SPDXVersion: SPDX-2.3\n")
	b.WriteString("DataLicense: CC0-1.0\n")
	b.WriteString("SPDXID: SPDXRef-DOCUMENT\n")
	b.WriteString(fmt.Sprintf("DocumentName: celador-scan-%s\n", time.Now().UTC().Format("20060102")))
	b.WriteString(fmt.Sprintf("DocumentNamespace: %s\n", ns))
	b.WriteString("\n")

	// Creation info
	b.WriteString("## Creation Information\n")
	b.WriteString("LicenseListVersion: 3.19\n")
	b.WriteString(fmt.Sprintf("Creator: Tool: Celador-%s\n", toolVersion))
	b.WriteString(fmt.Sprintf("Created: %s\n", time.Now().UTC().Format("2006-01-02T15:04:05Z")))
	b.WriteString("\n")

	// Packages
	for _, dep := range deps {
		pkgID := sanitizeSPDXID(dep.Name)
		purl := fmt.Sprintf("pkg:npm/%s@%s", dep.Name, dep.Version)
		downloadURL := fmt.Sprintf("https://registry.npmjs.org/%s/-/%s-%s.tgz", dep.Name, dep.Name, dep.Version)

		b.WriteString("## Package\n")
		b.WriteString(fmt.Sprintf("PackageName: %s\n", dep.Name))
		b.WriteString(fmt.Sprintf("SPDXID: SPDXRef-Package-%s\n", pkgID))
		b.WriteString(fmt.Sprintf("PackageVersion: %s\n", dep.Version))
		b.WriteString("PackageSupplier: NOASSERTION\n")
		b.WriteString(fmt.Sprintf("PackageDownloadLocation: %s\n", downloadURL))
		b.WriteString(fmt.Sprintf("ExternalRef: PACKAGE-MANAGER purl %s\n", purl))
		b.WriteString("PackageLicenseConcluded: NOASSERTION\n")
		b.WriteString("PackageLicenseInfoFromFiles: NOASSERTION\n")
		b.WriteString("PackageCopyrightText: NOASSERTION\n")
		b.WriteString("\n")
	}

	// Relationships
	b.WriteString("## Relationships\n")
	for _, dep := range deps {
		pkgID := sanitizeSPDXID(dep.Name)
		b.WriteString(fmt.Sprintf("Relationship: SPDXRef-DOCUMENT DESCRIBES SPDXRef-Package-%s\n", pkgID))
	}

	return []byte(b.String())
}

func sanitizeSPDXID(name string) string {
	var b strings.Builder
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.' {
			b.WriteRune(ch)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}
