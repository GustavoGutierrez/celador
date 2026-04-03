package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	fsadapter "github.com/GustavoGutierrez/celador/internal/adapters/fs"
	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type Result struct {
	Workspace shared.Workspace
	Messages  []string
}

type Service struct {
	fs       ports.FileSystem
	detector ports.WorkspaceDetector
	ignore   ports.IgnoreStore
	ui       ports.PromptUI
	node     ports.NodeVersionDetector
}

var strictNodeVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func NewService(fs ports.FileSystem, detector ports.WorkspaceDetector, ignore ports.IgnoreStore, ui ports.PromptUI, node ports.NodeVersionDetector) *Service {
	return &Service{fs: fs, detector: detector, ignore: ignore, ui: ui, node: node}
}

func (s *Service) Run(ctx context.Context, root string, tty bool, ci bool, installHook bool) (Result, error) {
	ws, err := s.detector.Detect(ctx, root, tty, ci)
	if err != nil {
		return Result{}, err
	}
	if ws.PackageManager == shared.PackageManagerUnknown {
		return Result{}, fmt.Errorf("unsupported workspace: no supported lockfile found")
	}

	messages := []string{}
	if err := s.ensureCeladorConfig(ctx, ws); err != nil {
		return Result{}, err
	}
	messages = append(messages, "wrote .celador.yaml")
	if err := s.ensureIgnoreTemplate(ctx, ws); err != nil {
		return Result{}, err
	}
	messages = append(messages, "wrote .celadorignore")
	if err := s.ensureIgnoreFiles(ctx, ws); err != nil {
		return Result{}, err
	}
	messages = append(messages, "updated ignore hygiene files")
	if err := s.ensureManagerHardening(ctx, ws); err != nil {
		return Result{}, err
	}
	messages = append(messages, "updated package manager hardening config")
	if err := s.ensureGuidanceFiles(ctx, ws); err != nil {
		return Result{}, err
	}
	messages = append(messages, "updated managed guidance files")
	if err := s.validateEngines(ctx, ws); err != nil {
		return Result{}, err
	}
	messages = append(messages, "validated strict engines")

	if installHook {
		if err := s.installHook(ctx, ws); err != nil {
			return Result{}, err
		}
		messages = append(messages, "installed pre-commit hook")
	}

	return Result{Workspace: ws, Messages: messages}, nil
}

func (s *Service) ensureCeladorConfig(ctx context.Context, ws shared.Workspace) error {
	config := map[string]any{}
	body, err := s.fs.ReadFile(ctx, ws.ConfigPath)
	if err == nil && strings.TrimSpace(string(body)) != "" {
		if err := yaml.Unmarshal(body, &config); err != nil {
			return fmt.Errorf("parse %s: %w", ws.ConfigPath, err)
		}
	}
	mergeMap(config, map[string]any{
		"cache":  map[string]any{"ttl": "24h"},
		"rules":  map[string]any{"version": "v1"},
		"output": map[string]any{"plain_text": true},
	})
	formatted, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", ws.ConfigPath, err)
	}
	return s.fs.WriteFile(ctx, ws.ConfigPath, formatted)
}

func (s *Service) ensureIgnoreTemplate(ctx context.Context, ws shared.Workspace) error {
	path := filepath.Join(ws.Root, ".celadorignore")
	if _, err := s.ignore.Load(ctx, ws.Root); err != nil {
		return err
	}
	defaultContent := "# finding-id|reason|expires-at(YYYY-MM-DD)\n"
	return fsadapter.WriteIfMissing(ctx, s.fs, path, []byte(defaultContent))
}

func (s *Service) ensureIgnoreFiles(ctx context.Context, ws shared.Workspace) error {
	requiredEntries := map[string][]string{
		".gitignore": {".env.local", "*.map.js", "*.js.map", "coverage/", ".celador/"},
		".npmignore": {".env.local", "*.map.js", "*.js.map", "coverage/"},
	}
	for name, entries := range requiredEntries {
		path := filepath.Join(ws.Root, name)
		content, _ := s.fs.ReadFile(ctx, path)
		lines := string(content)
		for _, required := range entries {
			if !strings.Contains(lines, required) {
				if len(lines) > 0 && !strings.HasSuffix(lines, "\n") {
					lines += "\n"
				}
				lines += required + "\n"
			}
		}
		if err := s.fs.WriteFile(ctx, path, []byte(lines)); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

func (s *Service) ensureManagerHardening(ctx context.Context, ws shared.Workspace) error {
	var path string
	switch ws.PackageManager {
	case shared.PackageManagerBun:
		path = filepath.Join(ws.Root, "bunfig.toml")
		return s.ensureBunHardening(ctx, path)
	case shared.PackageManagerDeno:
		path = filepath.Join(ws.Root, "deno.json")
		return s.ensureDenoHardening(ctx, path)
	default:
		path = filepath.Join(ws.Root, ".npmrc")
		return s.ensureNPMHardening(ctx, path)
	}
}

func (s *Service) ensureGuidanceFiles(ctx context.Context, ws shared.Workspace) error {
	managedBlock := `<!-- celador:start -->
## Celador Supply Chain Security
This project has been hardened against supply chain attacks using Celador.

### Rules for AI assistants and contributors
- Never use ^ or ~ in dependency version specifiers. Always pin exact versions.
- Always commit the lockfile. Never delete it or add it to .gitignore.
- Install scripts are disabled unless explicitly approved.
- New package versions must be at least 24 hours old.
- No dynamic Tailwind classes in arbitrary values.
- No raw SQL interpolation.
<!-- celador:end -->
`
	agentsPath := filepath.Join(ws.Root, "AGENTS.md")
	if err := fsadapter.WriteManagedBlock(ctx, s.fs, agentsPath, managedBlock); err != nil {
		return err
	}
	claudePath := filepath.Join(ws.Root, "CLAUDE.md")
	if exists, err := s.fs.Stat(ctx, claudePath); err != nil {
		return fmt.Errorf("stat CLAUDE.md: %w", err)
	} else if exists {
		if err := fsadapter.WriteManagedBlock(ctx, s.fs, claudePath, managedBlock); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateEngines(ctx context.Context, ws shared.Workspace) error {
	if ws.ManifestPath == "" {
		return nil
	}
	body, err := s.fs.ReadFile(ctx, ws.ManifestPath)
	if err != nil {
		return err
	}
	pkg := map[string]any{}
	if err := json.Unmarshal(body, &pkg); err != nil {
		return fmt.Errorf("parse package.json: %w", err)
	}
	enginesValue, hasEngines := pkg["engines"]
	if !hasEngines || enginesValue == nil {
		return s.ensureMissingNodeEngine(ctx, ws, pkg)
	}
	engines, ok := enginesValue.(map[string]any)
	if !ok {
		return fmt.Errorf("package.json engines must be an object with a strict engines.node entry such as \"20.11.1\"")
	}
	nodeValue, hasNode := engines["node"]
	if !hasNode || strings.TrimSpace(fmt.Sprint(nodeValue)) == "" {
		return s.ensureMissingNodeEngine(ctx, ws, pkg)
	}
	nodeVersion, ok := nodeValue.(string)
	if !ok || !isStrictNodeVersion(nodeVersion) {
		return fmt.Errorf("package.json engines.node must be a strict exact version such as \"20.11.1\", got %q", nodeValue)
	}
	return nil
}

func (s *Service) ensureMissingNodeEngine(ctx context.Context, ws shared.Workspace, pkg map[string]any) error {
	detected, ok := s.detectNodeVersion(ctx, ws.Root)
	if ws.CI || !ws.TTY {
		if !ok {
			return fmt.Errorf("package.json must define engines.node as a strict exact version such as \"20.11.1\"; unable to detect the current Node.js version automatically")
		}
		return s.writeNodeEngine(ctx, ws.ManifestPath, pkg, detected)
	}
	if !ok {
		return fmt.Errorf("package.json must define engines.node as a strict exact version such as \"20.11.1\"; Celador could not detect the current Node.js version automatically")
	}
	confirmed, err := s.ui.Confirm(ctx, fmt.Sprintf("package.json is missing engines.node. Add %s automatically using the current Node.js version?", detected))
	if err != nil {
		return fmt.Errorf("prompt to add package.json engines.node: %w", err)
	}
	if !confirmed {
		return fmt.Errorf("package.json must define engines.node as a strict exact version such as %q", detected)
	}
	return s.writeNodeEngine(ctx, ws.ManifestPath, pkg, detected)
}

func (s *Service) detectNodeVersion(ctx context.Context, root string) (string, bool) {
	if s.node == nil {
		return "", false
	}
	version, ok := s.node.Detect(ctx, root)
	if !ok || !isStrictNodeVersion(version) {
		return "", false
	}
	return version, true
}

func (s *Service) writeNodeEngine(ctx context.Context, path string, pkg map[string]any, version string) error {
	engines, ok := pkg["engines"].(map[string]any)
	if !ok || engines == nil {
		engines = map[string]any{}
	}
	engines["node"] = version
	pkg["engines"] = engines
	formatted, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}
	if err := s.fs.WriteFile(ctx, path, append(formatted, '\n')); err != nil {
		return fmt.Errorf("write package.json: %w", err)
	}
	return nil
}

func isStrictNodeVersion(value string) bool {
	return strictNodeVersionPattern.MatchString(strings.TrimSpace(value))
}

func (s *Service) installHook(ctx context.Context, ws shared.Workspace) error {
	hookPath := filepath.Join(ws.Root, ".git", "hooks", "pre-commit")
	content := "#!/bin/sh\ncelador scan\n"
	return s.fs.WriteFile(ctx, hookPath, []byte(content))
}

func (s *Service) ensureNPMHardening(ctx context.Context, path string) error {
	body, _ := s.fs.ReadFile(ctx, path)
	settings := map[string]string{
		"ignore-scripts":      "true",
		"save-exact":          "true",
		"trust-policy":        "no-downgrade",
		"minimum-release-age": "1440",
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	seen := map[string]bool{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || !strings.Contains(trimmed, "=") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		key := strings.TrimSpace(parts[0])
		value, ok := settings[key]
		if !ok {
			continue
		}
		lines[i] = key + "=" + value
		seen[key] = true
	}
	keys := sortedKeys(settings)
	for _, key := range keys {
		if seen[key] {
			continue
		}
		lines = append(lines, key+"="+settings[key])
	}
	return s.fs.WriteFile(ctx, path, []byte(strings.Join(lines, "\n")+"\n"))
}

func (s *Service) ensureBunHardening(ctx context.Context, path string) error {
	config := map[string]any{}
	body, _ := s.fs.ReadFile(ctx, path)
	if len(body) > 0 {
		if err := toml.Unmarshal(body, &config); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	}
	mergeMap(config, map[string]any{
		"install": map[string]any{
			"saveExact":                 true,
			"minimumReleaseAge":         1440,
			"minimumReleaseAgeExcludes": []any{"webpack", "react", "typescript", "vite", "next", "nuxt"},
		},
	})
	formatted, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	return s.fs.WriteFile(ctx, path, formatted)
}

func (s *Service) ensureDenoHardening(ctx context.Context, path string) error {
	config := map[string]any{}
	body, _ := s.fs.ReadFile(ctx, path)
	if len(body) > 0 {
		if err := json.Unmarshal(body, &config); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	}
	config["lock"] = true
	formatted, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	return s.fs.WriteFile(ctx, path, append(formatted, '\n'))
}

func mergeMap(dst map[string]any, src map[string]any) {
	for key, value := range src {
		srcMap, srcIsMap := value.(map[string]any)
		dstMap, dstIsMap := dst[key].(map[string]any)
		if srcIsMap {
			if !dstIsMap {
				dstMap = map[string]any{}
			}
			mergeMap(dstMap, srcMap)
			dst[key] = dstMap
			continue
		}
		dst[key] = value
	}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
