package provenance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Result represents the outcome of a provenance verification check.
type Result struct {
	PackageName    string
	Version        string
	HasAttestation bool
	SignerIdentity string
	BuildURI       string
	Verified       bool
	Warning        string
	RegistryURL    string
}

// Checker verifies npm package provenance using registry attestations.
type Checker struct {
	client      *http.Client
	npmRegistry string
}

// NewChecker creates a provenance checker that reads the registry URL from
// the CELADOR_NPM_REGISTRY environment variable (falls back to the official
// npm registry) and an optional override URL passed at runtime.
func NewChecker() *Checker {
	return NewCheckerWithEndpoint("")
}

// NewCheckerWithEndpoint allows overriding the npm registry URL for testing
// or enterprise use-cases (AWS CodeArtifact, GitHub Packages, Verdaccio, etc.).
// If overrideURL is empty the checker falls back to CELADOR_NPM_REGISTRY or the
// official npm registry.
func NewCheckerWithEndpoint(overrideURL string) *Checker {
	registry := officialRegistry()
	if overrideURL != "" {
		registry = overrideURL
	}
	return &Checker{
		client:      &http.Client{Timeout: 10 * time.Second},
		npmRegistry: strings.TrimRight(registry, "/"),
	}
}

// officialRegistry returns the configured registry URL.
func officialRegistry() string {
	if env := os.Getenv("CELADOR_NPM_REGISTRY"); env != "" {
		return env
	}
	return "https://registry.npmjs.org"
}

// CheckPackage verifies the provenance of an npm package by checking
// its attestation statement in the registry metadata.
func (c *Checker) CheckPackage(ctx context.Context, name, version string) (Result, error) {
	result := Result{
		PackageName: name,
		Version:     version,
		RegistryURL: c.npmRegistry,
	}

	// Enterprise registries (AWS CodeArtifact, GitHub Packages, etc.) may not
	// support the /{name}/{version} endpoint. We try it first and gracefully
	// handle 404 / 403 as "provenance unavailable" rather than a hard error.
	url := fmt.Sprintf("%s/%s/%s", c.npmRegistry, name, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		result.Warning = fmt.Sprintf("provenance not available for %s@%s on %s", name, version, c.npmRegistry)
		return result, nil
	}

	if resp.StatusCode >= 300 {
		return result, fmt.Errorf("registry returned %s for %s@%s", resp.Status, name, version)
	}

	var metadata map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return result, fmt.Errorf("decode registry metadata: %w", err)
	}

	dist, _ := metadata["dist"].(map[string]any)
	if dist == nil {
		result.Warning = "no dist metadata found"
		return result, nil
	}

	attestations, _ := dist["attestations"].(map[string]any)
	if attestations == nil {
		result.Warning = fmt.Sprintf("no provenance statement for %s@%s", name, version)
		return result, nil
	}

	// Has attestations — check the signer identity
	provenance, _ := attestations["provenance"].(map[string]any)
	if provenance != nil {
		result.SignerIdentity = extractSignerIdentity(provenance)
		if buildURI, _ := provenance["buildType"].(string); buildURI != "" {
			result.BuildURI = buildURI
		}
	}

	// Only consider verified if signer is known (GitHub Actions or other
	// recognized CI systems that support SLSA provenance).
	if result.SignerIdentity != "unknown" {
		result.HasAttestation = true
		result.Verified = true
	} else {
		result.Warning = fmt.Sprintf("provenance from unrecognized CI on %s", c.npmRegistry)
	}

	return result, nil
}

// extractSignerIdentity determines the CI system that signed the provenance.
func extractSignerIdentity(provenance map[string]any) string {
	buildType, _ := provenance["buildType"].(string)
	if buildType == "" {
		return "unknown"
	}
	lower := strings.ToLower(buildType)
	if strings.Contains(lower, "github") || strings.Contains(lower, "actions") {
		return "GitHub Actions"
	}
	if strings.Contains(lower, "gitlab") {
		return "GitLab CI"
	}
	if strings.Contains(lower, "jenkins") {
		return "Jenkins"
	}
	if strings.Contains(lower, "circle") {
		return "CircleCI"
	}
	if strings.Contains(lower, "azure") || strings.Contains(lower, "devops") {
		return "Azure DevOps"
	}
	if strings.Contains(lower, "aws") || strings.Contains(lower, "codebuild") {
		return "AWS CodeBuild"
	}
	return "unknown"
}
