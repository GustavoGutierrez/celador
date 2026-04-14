package osv

import (
	"archive/tar"
	"io"
	"path/filepath"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

// sourceFileExtensions defines which files inside a tarball should be
// inspected for malicious patterns. package.json is always included.
var sourceFileExtensions = map[string]bool{
	".js":   true,
	".ts":   true,
	".mjs":  true,
	".cjs":  true,
	".node": true,
}

// maliciousPattern defines a pattern to search for in source files.
type maliciousPattern struct {
	name     string
	pattern  string
	severity shared.Severity
}

// knownMaliciousPatterns is the list of patterns that indicate potentially
// malicious behaviour in package source files.
var knownMaliciousPatterns = []maliciousPattern{
	{name: "eval() dynamic execution", pattern: "eval(", severity: shared.SeverityHigh},
	{name: "new Function() dynamic execution", pattern: "new function(", severity: shared.SeverityHigh},
	{name: "child_process.exec", pattern: "child_process", severity: shared.SeverityHigh},
	{name: "process.env with network activity", pattern: "process.env", severity: shared.SeverityHigh},
	{name: "network exfiltration (https)", pattern: "https.request", severity: shared.SeverityMedium},
	{name: "network exfiltration (http)", pattern: "http.request", severity: shared.SeverityMedium},
	{name: "network exfiltration (fetch)", pattern: "fetch(", severity: shared.SeverityMedium},
}

// isSourceFile returns true if the file path should be inspected for
// malicious patterns. package.json is always considered a source file.
func isSourceFile(name string) bool {
	base := filepath.Base(name)
	if base == "package.json" {
		return true
	}
	ext := filepath.Ext(base)
	return sourceFileExtensions[ext]
}

// inspectSourceFile reads a single file from the tar archive and checks
// it against known malicious patterns.
func (r *RegistryInspector) inspectSourceFile(tr *tar.Reader, name string, assessment *shared.InstallAssessment) error {
	body, err := io.ReadAll(io.LimitReader(tr, 1<<20))
	if err != nil {
		return err
	}
	text := strings.ToLower(string(body))

	// Package.json gets specialized checks (lifecycle scripts, env+network)
	if filepath.Base(name) == "package.json" {
		if strings.Contains(text, "scripts") && (strings.Contains(text, "postinstall") ||
			strings.Contains(text, "preinstall") ||
			strings.Contains(text, `"prepare"`) ||
			strings.Contains(text, `"install"`)) {
			assessment.Risk = maxSeverity(assessment.Risk, shared.SeverityMedium)
			assessment.ShouldPrompt = true
			assessment.Reasons = append(assessment.Reasons, "package defines install-time scripts")
		}
		if strings.Contains(text, "process.env") && (strings.Contains(text, "http://") || strings.Contains(text, "https://") || strings.Contains(text, "fetch(")) {
			assessment.Risk = maxSeverity(assessment.Risk, shared.SeverityHigh)
			assessment.ShouldPrompt = true
			assessment.Reasons = append(assessment.Reasons, "package scripts reference env data and network activity")
		}
	}

	// Check .node binaries
	if filepath.Ext(name) == ".node" {
		assessment.Risk = maxSeverity(assessment.Risk, shared.SeverityMedium)
		assessment.ShouldPrompt = true
		assessment.Reasons = append(assessment.Reasons, "package contains native .node binary")
		return nil
	}

	// Check for known malicious patterns in source files
	for _, mp := range knownMaliciousPatterns {
		if strings.Contains(text, mp.pattern) {
			assessment.Risk = maxSeverity(assessment.Risk, mp.severity)
			assessment.ShouldPrompt = true
			assessment.Reasons = append(assessment.Reasons, mp.name+" detected in "+filepath.Base(name))
		}
	}

	// Check for long hex-encoded strings in source files
	// Scan character-by-character for contiguous hex sequences
	hexRun := strings.Builder{}
	for _, ch := range text {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			hexRun.WriteRune(ch)
		} else {
			if hexRun.Len() > 80 {
				assessment.Risk = maxSeverity(assessment.Risk, shared.SeverityMedium)
				assessment.ShouldPrompt = true
				assessment.Reasons = append(assessment.Reasons, "obfuscated string detected in "+filepath.Base(name))
				break
			}
			hexRun.Reset()
		}
	}

	return nil
}
