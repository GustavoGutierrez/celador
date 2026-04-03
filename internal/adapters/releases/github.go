package releases

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GitHubLatestReleaseSource struct {
	client   *http.Client
	endpoint string
}

func NewGitHubLatestReleaseSource() *GitHubLatestReleaseSource {
	return &GitHubLatestReleaseSource{
		client:   &http.Client{Timeout: 3 * time.Second},
		endpoint: "https://api.github.com/repos/GustavoGutierrez/celador/releases/latest",
	}
}

func (s *GitHubLatestReleaseSource) Latest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create latest release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "celador-version-check")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("latest release lookup returned %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode latest release payload: %w", err)
	}
	return payload.TagName, nil
}
