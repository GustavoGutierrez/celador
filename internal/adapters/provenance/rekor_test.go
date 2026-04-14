package provenance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestChecker_PackageWithAttestation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"name": "express",
			"dist": {
				"attestations": {
					"url": "https://registry.npmjs.org/-/npm/v1/attestations/express@4.18.2",
					"provenance": {
						"predicateType": "https://slsa.dev/provenance/v0.2",
						"buildType": "https://actions.github.io/buildtypes/workflow/v1"
					}
				}
			}
		}`))
	}))
	defer server.Close()

	checker := NewCheckerWithEndpoint(server.URL)
	result, err := checker.CheckPackage(context.Background(), "express", "4.18.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasAttestation {
		t.Error("expected HasAttestation=true for package with provenance")
	}
	if result.SignerIdentity != "GitHub Actions" {
		t.Errorf("expected 'GitHub Actions' signer, got %q", result.SignerIdentity)
	}
	if !result.Verified {
		t.Error("expected Verified=true for valid attestation")
	}
}

func TestChecker_PackageWithoutAttestation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name": "old-pkg","dist":{"tarball":"https://example.com/pkg.tgz"}}`))
	}))
	defer server.Close()

	checker := NewCheckerWithEndpoint(server.URL)
	result, err := checker.CheckPackage(context.Background(), "old-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasAttestation {
		t.Error("expected HasAttestation=false for package without provenance")
	}
	if result.Warning == "" {
		t.Error("expected warning message for missing provenance")
	}
	if !strings.Contains(result.Warning, "no provenance") {
		t.Errorf("expected 'no provenance' in warning, got %q", result.Warning)
	}
}

func TestChecker_UnknownCISigner(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"name": "suspicious-pkg",
			"dist": {
				"attestations": {
					"url": "https://example.com/attestations",
					"provenance": {
						"predicateType": "https://slsa.dev/provenance/v0.2",
						"buildType": "unknown-ci"
					}
				}
			}
		}`))
	}))
	defer server.Close()

	checker := NewCheckerWithEndpoint(server.URL)
	result, err := checker.CheckPackage(context.Background(), "suspicious-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SignerIdentity != "unknown" {
		t.Errorf("expected 'unknown' signer, got %q", result.SignerIdentity)
	}
	if result.HasAttestation {
		t.Error("expected HasAttestation=false for unknown signer")
	}
}

func TestChecker_NetworkError(t *testing.T) {
	t.Parallel()
	checker := NewCheckerWithEndpoint("http://127.0.0.1:1")
	_, err := checker.CheckPackage(context.Background(), "pkg", "1.0.0")
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}

func TestChecker_PackageNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewCheckerWithEndpoint(server.URL)
	result, err := checker.CheckPackage(context.Background(), "nonexistent", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasAttestation {
		t.Error("expected HasAttestation=false for non-existent package")
	}
}

func TestChecker_ForbiddenResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	checker := NewCheckerWithEndpoint(server.URL)
	result, err := checker.CheckPackage(context.Background(), "private-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error for 403: %v", err)
	}
	if result.HasAttestation {
		t.Error("expected HasAttestation=false for forbidden registry")
	}
}

func TestExtractSignerIdentity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		json     string
		expected string
	}{
		{"GitHub Actions SLSA", `{"buildType":"https://actions.github.io/buildtypes/workflow/v1"}`, "GitHub Actions"},
		{"GitHub Actions generic", `{"buildType":"https://github.com/actions/workflow"}`, "GitHub Actions"},
		{"GitLab CI", `{"buildType":"https://gitlab.com/ci"}`, "GitLab CI"},
		{"Jenkins", `{"buildType":"https://jenkins.io/provenance"}`, "Jenkins"},
		{"CircleCI", `{"buildType":"https://circleci.com/build"}`, "CircleCI"},
		{"Azure DevOps", `{"buildType":"https://dev.azure.com/build"}`, "Azure DevOps"},
		{"AWS CodeBuild", `{"buildType":"https://aws.amazon.com/codebuild"}`, "AWS CodeBuild"},
		{"Empty", `{}`, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provenance map[string]any
			json.Unmarshal([]byte(tt.json), &provenance)
			result := extractSignerIdentity(provenance)
			if result != tt.expected {
				t.Errorf("extractSignerIdentity(%s) = %q, want %q", tt.json, result, tt.expected)
			}
		})
	}
}

func TestNewChecker_Defaults(t *testing.T) {
	t.Parallel()
	checker := NewChecker()
	if checker.client == nil {
		t.Error("expected non-nil http.Client")
	}
	if !strings.Contains(checker.npmRegistry, "registry.npmjs.org") {
		t.Errorf("expected npm registry URL, got %q", checker.npmRegistry)
	}
}

func TestNewChecker_EnvOverride(t *testing.T) {
	t.Parallel()
	original := os.Getenv("CELADOR_NPM_REGISTRY")
	os.Setenv("CELADOR_NPM_REGISTRY", "https://my-enterprise-registry.com")
	defer os.Setenv("CELADOR_NPM_REGISTRY", original)

	checker := NewChecker()
	if !strings.Contains(checker.npmRegistry, "my-enterprise-registry.com") {
		t.Errorf("expected enterprise registry, got %q", checker.npmRegistry)
	}
}

func TestNewCheckerWithEndpoint_Override(t *testing.T) {
	t.Parallel()
	checker := NewCheckerWithEndpoint("https://override.registry.com")
	if checker.npmRegistry != "https://override.registry.com" {
		t.Errorf("expected override URL, got %q", checker.npmRegistry)
	}
}

func TestChecker_MalformedResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not valid json{{{`))
	}))
	defer server.Close()

	checker := NewCheckerWithEndpoint(server.URL)
	_, err := checker.CheckPackage(context.Background(), "pkg", "1.0.0")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestChecker_RegistryURLInResult(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"pkg","dist":{}}`))
	}))
	defer server.Close()

	checker := NewCheckerWithEndpoint(server.URL)
	result, err := checker.CheckPackage(context.Background(), "pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.RegistryURL, server.URL) {
		t.Errorf("expected registry URL in result, got %q", result.RegistryURL)
	}
}
