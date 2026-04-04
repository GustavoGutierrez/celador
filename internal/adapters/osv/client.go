package osv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type Client struct {
	httpClient *http.Client
	ttl        time.Duration
	endpoint   string
	vulnAPI    string
}

func NewClient(ttl time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 20 * time.Second},
		ttl:        ttl,
		endpoint:   "https://api.osv.dev/v1/querybatch",
		vulnAPI:    "https://api.osv.dev/v1/vulns",
	}
}

type osvAdvisory struct {
	ID       string        `json:"id"`
	Summary  string        `json:"summary"`
	Details  string        `json:"details"`
	Aliases  []string      `json:"aliases"`
	Affected []osvAffected `json:"affected"`
}

type osvAffected struct {
	Package struct {
		Name string `json:"name"`
	} `json:"package"`
	Ranges []osvRange `json:"ranges"`
}

type osvRange struct {
	Type   string              `json:"type"`
	Events []map[string]string `json:"events"`
}

func (c *Client) Query(ctx context.Context, deps []shared.Dependency) ([]shared.Finding, error) {
	queries := make([]map[string]any, 0, len(deps))
	for _, dep := range deps {
		queries = append(queries, map[string]any{
			"package": map[string]string{"name": dep.Name, "ecosystem": dep.Ecosystem},
			"version": dep.Version,
		})
	}
	body, err := json.Marshal(map[string]any{"queries": queries})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("osv returned %s", resp.Status)
	}
	var decoded struct {
		Results []struct {
			Vulns []osvAdvisory `json:"vulns"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	findings := []shared.Finding{}
	hydrated := map[string]osvAdvisory{}
	for i, result := range decoded.Results {
		dep := deps[i]
		for _, vuln := range result.Vulns {
			advisory := vuln
			if advisoryNeedsHydration(advisory) {
				if cached, ok := hydrated[advisory.ID]; ok {
					advisory = cached
				} else if details, err := c.fetchAdvisory(ctx, advisory.ID); err == nil {
					hydrated[advisory.ID] = details
					advisory = details
				}
			}
			fixVersion := fixedVersionForPackage(advisory, dep.Name, dep.Version)
			finding := shared.Finding{
				ID:          advisory.ID,
				Source:      shared.FindingSourceOSV,
				Severity:    shared.SeverityHigh,
				Target:      dep.Name,
				PackageName: dep.Name,
				Summary:     summarizeVulnerability(advisory.Summary, advisory.Details, dep.Name),
				FixVersion:  fixVersion,
				Fixable:     fixVersion != "",
			}
			findings = append(findings, finding)
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].ID == findings[j].ID {
			return findings[i].PackageName < findings[j].PackageName
		}
		return findings[i].ID < findings[j].ID
	})
	return findings, nil
}

func (c *Client) fetchAdvisory(ctx context.Context, id string) (osvAdvisory, error) {
	url := strings.TrimRight(c.vulnAPI, "/") + "/" + url.PathEscape(strings.TrimSpace(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return osvAdvisory{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return osvAdvisory{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return osvAdvisory{}, fmt.Errorf("osv advisory %s returned %s: %s", id, resp.Status, strings.TrimSpace(string(body)))
	}
	var advisory osvAdvisory
	if err := json.NewDecoder(resp.Body).Decode(&advisory); err != nil {
		return osvAdvisory{}, err
	}
	return advisory, nil
}

func advisoryNeedsHydration(vuln osvAdvisory) bool {
	return len(vuln.Affected) == 0
}

func fixedVersionForPackage(vuln osvAdvisory, packageName string, currentVersion string) string {
	packageName = strings.TrimSpace(packageName)
	best := ""
	fallback := ""
	for _, affected := range vuln.Affected {
		if strings.TrimSpace(affected.Package.Name) != packageName {
			continue
		}
		for _, r := range affected.Ranges {
			if candidate := fixedVersionForRange(r, currentVersion); candidate != "" {
				if best == "" || shared.CompareVersions(candidate, best) < 0 {
					best = candidate
				}
			}
			if candidate := earliestFixedVersion(r.Events); candidate != "" {
				if fallback == "" || shared.CompareVersions(candidate, fallback) < 0 {
					fallback = candidate
				}
			}
		}
	}
	if best != "" {
		return best
	}
	return fallback
}

func fixedVersionForRange(r osvRange, currentVersion string) string {
	if current := shared.NormalizeVersion(currentVersion); current == "" || (strings.TrimSpace(r.Type) != "" && !strings.EqualFold(r.Type, "SEMVER")) {
		return ""
	}
	active := false
	for _, event := range r.Events {
		if introduced, ok := event["introduced"]; ok {
			introduced = strings.TrimSpace(introduced)
			if introduced == "0" || shared.CompareVersions(currentVersion, introduced) >= 0 {
				active = true
			} else {
				active = false
			}
		}
		if fixed, ok := event["fixed"]; ok {
			fixed = strings.TrimSpace(fixed)
			if active && shared.CompareVersions(currentVersion, fixed) < 0 {
				return fixed
			}
			if shared.CompareVersions(currentVersion, fixed) >= 0 {
				active = false
			}
		}
		if lastAffected, ok := event["last_affected"]; ok {
			lastAffected = strings.TrimSpace(lastAffected)
			if active && shared.CompareVersions(currentVersion, lastAffected) <= 0 {
				return ""
			}
			if shared.CompareVersions(currentVersion, lastAffected) > 0 {
				active = false
			}
		}
		if limit, ok := event["limit"]; ok {
			limit = strings.TrimSpace(limit)
			if active && shared.CompareVersions(currentVersion, limit) < 0 {
				return ""
			}
			if shared.CompareVersions(currentVersion, limit) >= 0 {
				active = false
			}
		}
	}
	return ""
}

func earliestFixedVersion(events []map[string]string) string {
	best := ""
	for _, event := range events {
		fixed := strings.TrimSpace(event["fixed"])
		if fixed == "" {
			continue
		}
		if best == "" || shared.CompareVersions(fixed, best) < 0 {
			best = fixed
		}
	}
	return best
}

func summarizeVulnerability(summary string, details string, packageName string) string {
	summary = strings.TrimSpace(summary)
	details = firstAdvisorySentence(details)
	if summary != "" && !(details != "" && isGenericVulnerabilitySummary(summary, packageName)) {
		return summary
	}

	if details != "" {
		return details
	}

	packageName = strings.TrimSpace(packageName)
	if packageName != "" {
		return fmt.Sprintf("Vulnerability detected in %s", packageName)
	}

	return "Vulnerability detected"
}

func compactWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func firstAdvisorySentence(value string) string {
	value = compactWhitespace(value)
	if value == "" {
		return ""
	}

	for _, separator := range []string{". ", "! ", "? "} {
		if idx := strings.Index(value, separator); idx >= 0 {
			return strings.TrimSpace(value[:idx+1])
		}
	}

	return value
}

func isGenericVulnerabilitySummary(summary string, packageName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(summary))
	if normalized == "" {
		return true
	}

	packageName = strings.ToLower(strings.TrimSpace(packageName))
	generic := []string{
		"vulnerability detected",
		"security vulnerability",
		"known vulnerability",
	}
	if packageName != "" {
		generic = append(generic,
			fmt.Sprintf("vulnerability detected in %s", packageName),
			fmt.Sprintf("vulnerability in %s", packageName),
			fmt.Sprintf("security vulnerability in %s", packageName),
		)
	}

	for _, candidate := range generic {
		if normalized == candidate {
			return true
		}
	}

	return false
}
