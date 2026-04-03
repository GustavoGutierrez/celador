package osv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type Client struct {
	httpClient *http.Client
	ttl        time.Duration
}

func NewClient(ttl time.Duration) *Client {
	return &Client{httpClient: &http.Client{Timeout: 20 * time.Second}, ttl: ttl}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.osv.dev/v1/querybatch", bytes.NewReader(body))
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
			Vulns []struct {
				ID       string   `json:"id"`
				Summary  string   `json:"summary"`
				Aliases  []string `json:"aliases"`
				Affected []struct {
					Ranges []struct {
						Events []map[string]string `json:"events"`
					} `json:"ranges"`
				} `json:"affected"`
			} `json:"vulns"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	findings := []shared.Finding{}
	for i, result := range decoded.Results {
		for _, vuln := range result.Vulns {
			fixVersion := ""
			for _, affected := range vuln.Affected {
				for _, r := range affected.Ranges {
					for _, event := range r.Events {
						if fixed, ok := event["fixed"]; ok && (fixVersion == "" || strings.Compare(fixed, fixVersion) < 0) {
							fixVersion = fixed
						}
					}
				}
			}
			finding := shared.Finding{
				ID:          vuln.ID,
				Source:      shared.FindingSourceOSV,
				Severity:    shared.SeverityHigh,
				Target:      deps[i].Name,
				PackageName: deps[i].Name,
				Summary:     vuln.Summary,
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
