package fs

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type IgnoreStore struct{ fs ports.FileSystem }

func NewIgnoreStore(fs ports.FileSystem) *IgnoreStore { return &IgnoreStore{fs: fs} }

func (s *IgnoreStore) Load(ctx context.Context, root string) ([]shared.IgnoreRule, error) {
	path := filepath.Join(root, ".celadorignore")
	body, err := s.fs.ReadFile(ctx, path)
	if err != nil {
		if ok, statErr := s.fs.Stat(ctx, path); statErr == nil && !ok {
			return nil, nil
		}
		return nil, err
	}
	var rules []shared.IgnoreRule
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		rule := shared.IgnoreRule{Selector: parts[0]}
		if len(parts) > 1 {
			rule.Reason = parts[1]
		}
		if len(parts) > 2 && parts[2] != "" {
			if t, err := time.Parse("2006-01-02", parts[2]); err == nil {
				rule.ExpiresAt = &t
			}
		}
		rules = append(rules, rule)
	}
	return rules, nil
}
