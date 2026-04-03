package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
	"gopkg.in/yaml.v3"
)

type NPMParser struct{ fs ports.FileSystem }
type PNPMParser struct{ fs ports.FileSystem }
type BunParser struct{ fs ports.FileSystem }
type DenoParser struct{ fs ports.FileSystem }

func NewNPMParser(fs ports.FileSystem) *NPMParser   { return &NPMParser{fs: fs} }
func NewPNPMParser(fs ports.FileSystem) *PNPMParser { return &PNPMParser{fs: fs} }
func NewBunParser(fs ports.FileSystem) *BunParser   { return &BunParser{fs: fs} }
func NewDenoParser(fs ports.FileSystem) *DenoParser { return &DenoParser{fs: fs} }

func (p *NPMParser) FileSystem() ports.FileSystem  { return p.fs }
func (p *PNPMParser) FileSystem() ports.FileSystem { return p.fs }
func (p *BunParser) FileSystem() ports.FileSystem  { return p.fs }
func (p *DenoParser) FileSystem() ports.FileSystem { return p.fs }

func (p *NPMParser) Supports(path string) bool  { return filepath.Base(path) == "package-lock.json" }
func (p *PNPMParser) Supports(path string) bool { return filepath.Base(path) == "pnpm-lock.yaml" }
func (p *BunParser) Supports(path string) bool {
	return filepath.Base(path) == "bun.lock" || filepath.Base(path) == "bun.lockb"
}
func (p *DenoParser) Supports(path string) bool { return filepath.Base(path) == "deno.lock" }

func (p *NPMParser) Parse(ctx context.Context, workspace shared.Workspace, path string) ([]shared.Dependency, error) {
	body, err := p.fs.ReadFile(ctx, path)
	if err != nil {
		return nil, err
	}
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		return nil, err
	}
	deps := []shared.Dependency{}
	for key, value := range lock.Packages {
		if key == "" || !strings.Contains(key, "node_modules/") || value.Version == "" {
			continue
		}
		name := key[strings.LastIndex(key, "node_modules/")+13:]
		deps = append(deps, shared.Dependency{Name: name, Version: value.Version, Ecosystem: "npm", Direct: strings.Count(key, "node_modules/") == 1, Manifest: workspace.ManifestPath})
	}
	sortDeps(deps)
	return deps, nil
}

func (p *PNPMParser) Parse(ctx context.Context, workspace shared.Workspace, path string) ([]shared.Dependency, error) {
	body, err := p.fs.ReadFile(ctx, path)
	if err != nil {
		return nil, err
	}
	var lock struct {
		Packages map[string]map[string]any `yaml:"packages"`
	}
	if err := yaml.Unmarshal(body, &lock); err != nil {
		return nil, err
	}
	deps := []shared.Dependency{}
	for key := range lock.Packages {
		trimmed := strings.TrimPrefix(key, "/")
		parts := strings.Split(trimmed, "@")
		if len(parts) < 2 {
			continue
		}
		version := parts[len(parts)-1]
		name := strings.TrimSuffix(trimmed, "@"+version)
		deps = append(deps, shared.Dependency{Name: name, Version: version, Ecosystem: "npm", Manifest: workspace.ManifestPath})
	}
	sortDeps(deps)
	return deps, nil
}

func (p *BunParser) Parse(ctx context.Context, workspace shared.Workspace, path string) ([]shared.Dependency, error) {
	body, err := p.fs.ReadFile(ctx, path)
	if err != nil {
		return nil, err
	}
	deps := []shared.Dependency{}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		deps = append(deps, shared.Dependency{Name: fields[0], Version: fields[1], Ecosystem: "npm", Manifest: workspace.ManifestPath})
	}
	sortDeps(deps)
	return deps, nil
}

func (p *DenoParser) Parse(ctx context.Context, workspace shared.Workspace, path string) ([]shared.Dependency, error) {
	body, err := p.fs.ReadFile(ctx, path)
	if err != nil {
		return nil, err
	}
	var lock struct {
		Packages struct {
			Specifiers map[string]string `json:"specifiers"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		return nil, err
	}
	deps := []shared.Dependency{}
	for name, version := range lock.Packages.Specifiers {
		deps = append(deps, shared.Dependency{Name: name, Version: strings.TrimPrefix(version, "npm:"), Ecosystem: "npm", Manifest: workspace.ManifestPath})
	}
	sortDeps(deps)
	return deps, nil
}

func sortDeps(deps []shared.Dependency) {
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].Name == deps[j].Name {
			return deps[i].Version < deps[j].Version
		}
		return deps[i].Name < deps[j].Name
	})
}

func Fingerprint(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func readLockfileBody(ctx context.Context, fs ports.FileSystem, paths []string) (string, error) {
	var builder strings.Builder
	for _, path := range paths {
		body, err := fs.ReadFile(ctx, path)
		if err != nil {
			return "", fmt.Errorf("read lockfile %s: %w", path, err)
		}
		builder.WriteString(path)
		builder.WriteByte('\n')
		builder.Write(body)
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}
