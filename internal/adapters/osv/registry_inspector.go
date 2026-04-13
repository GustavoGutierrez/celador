package osv

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type RegistryInspector struct {
	client   *http.Client
	registry string
}

func NewRegistryInspector() *RegistryInspector {
	return &RegistryInspector{
		client:   &http.Client{Timeout: 20 * time.Second},
		registry: "https://registry.npmjs.org",
	}
}

// NewRegistryInspectorWithEndpoint allows overriding the registry URL for testing.
func NewRegistryInspectorWithEndpoint(registry string) *RegistryInspector {
	return &RegistryInspector{
		client:   &http.Client{Timeout: 20 * time.Second},
		registry: registry,
	}
}

func (r *RegistryInspector) InspectPackage(ctx context.Context, manager shared.PackageManager, pkg string) (shared.InstallAssessment, error) {
	assessment := shared.InstallAssessment{Package: pkg, Manager: manager, Risk: shared.SeverityLow}
	if manager == shared.PackageManagerDeno {
		assessment.Unknown = true
		assessment.ShouldPrompt = true
		assessment.Reasons = append(assessment.Reasons, "deno install handoff is not supported in v1")
		return assessment, nil
	}
	url := fmt.Sprintf("%s/%s/latest", r.registry, pkg)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return assessment, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		assessment.Unknown = true
		assessment.ShouldPrompt = true
		assessment.Risk = shared.SeverityMedium
		assessment.Reasons = append(assessment.Reasons, "package metadata could not be fetched")
		return assessment, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		assessment.Unknown = true
		assessment.ShouldPrompt = true
		assessment.Risk = shared.SeverityMedium
		assessment.Reasons = append(assessment.Reasons, "package metadata is unavailable")
		return assessment, nil
	}
	var meta struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Dist    struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return assessment, err
	}
	assessment.TarballURL = meta.Dist.Tarball
	if meta.Dist.Tarball == "" {
		assessment.Unknown = true
		assessment.ShouldPrompt = true
		assessment.Risk = shared.SeverityMedium
		assessment.Reasons = append(assessment.Reasons, "tarball URL is missing")
		return assessment, nil
	}
	if err := r.inspectTarball(ctx, &assessment); err != nil {
		assessment.Unknown = true
		assessment.ShouldPrompt = true
		assessment.Risk = shared.SeverityMedium
		assessment.Reasons = append(assessment.Reasons, "tarball inspection failed")
	}
	return assessment, nil
}

func (r *RegistryInspector) inspectTarball(ctx context.Context, assessment *shared.InstallAssessment) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assessment.TarballURL, nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if !strings.HasSuffix(header.Name, "package.json") {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(tr, 1<<20))
		if err != nil {
			return err
		}
		text := strings.ToLower(string(body))
		if strings.Contains(text, "process.env") && (strings.Contains(text, "http://") || strings.Contains(text, "https://") || strings.Contains(text, "fetch(")) {
			assessment.Risk = shared.SeverityHigh
			assessment.ShouldPrompt = true
			assessment.Reasons = append(assessment.Reasons, "package scripts reference env data and network activity")
		}
		if strings.Contains(text, "scripts") && strings.Contains(text, "postinstall") {
			assessment.Risk = maxSeverity(assessment.Risk, shared.SeverityMedium)
			assessment.ShouldPrompt = true
			assessment.Reasons = append(assessment.Reasons, "package defines install-time scripts")
		}
		for _, token := range strings.Fields(text) {
			if len(token) > 80 && isHexLike(token) {
				assessment.Risk = maxSeverity(assessment.Risk, shared.SeverityMedium)
				assessment.ShouldPrompt = true
				assessment.Reasons = append(assessment.Reasons, "package contains long encoded strings")
				break
			}
		}
		return nil
	}
}

func isHexLike(s string) bool {
	count := 0
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			count++
			continue
		}
		return false
	}
	return count == len(s)
}

func maxSeverity(a, b shared.Severity) shared.Severity {
	rank := map[shared.Severity]int{shared.SeverityLow: 1, shared.SeverityMedium: 2, shared.SeverityHigh: 3, shared.SeverityCritical: 4}
	if rank[b] > rank[a] {
		return b
	}
	return a
}
