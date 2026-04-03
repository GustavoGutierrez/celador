package rules

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
	"gopkg.in/yaml.v3"
)

type YAMLLoader struct{ fs ports.FileSystem }

func NewYAMLLoader(fs ports.FileSystem) *YAMLLoader { return &YAMLLoader{fs: fs} }

func (l *YAMLLoader) Load(ctx context.Context, root string) ([]shared.RuleConfig, string, error) {
	paths, err := l.fs.Glob(ctx, filepath.Join(root, "configs", "rules"), "*.yaml")
	if err != nil {
		return nil, "", err
	}
	var rules []shared.RuleConfig
	version := "v1"
	for _, path := range paths {
		body, err := l.fs.ReadFile(ctx, path)
		if err != nil {
			return nil, "", err
		}
		var pack shared.RulePack
		if err := yaml.Unmarshal(body, &pack); err != nil {
			return nil, "", fmt.Errorf("parse rule pack %s: %w", path, err)
		}
		if pack.Version != "" {
			version = pack.Version
		}
		rules = append(rules, pack.Rules...)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	return rules, version, nil
}
