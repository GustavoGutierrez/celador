package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type Detector struct{ fs ports.FileSystem }

func NewDetector(fs ports.FileSystem) *Detector { return &Detector{fs: fs} }

func (d *Detector) Detect(ctx context.Context, root string, tty bool, ci bool) (shared.Workspace, error) {
	ws := shared.Workspace{Root: root, PackageManager: shared.PackageManagerUnknown, TTY: tty, CI: ci, ConfigPath: filepath.Join(root, ".celador.yaml")}
	manifest := filepath.Join(root, "package.json")
	hints := map[shared.PackageManager]struct{}{}
	manifestExists := false
	hasUnsupportedExplicitManager := false
	if ok, err := d.fs.Stat(ctx, manifest); err != nil {
		return ws, fmt.Errorf("stat package.json: %w", err)
	} else if ok {
		manifestExists = true
		ws.ManifestPath = manifest
		pkgJSON, err := d.fs.ReadFile(ctx, manifest)
		if err != nil {
			return ws, fmt.Errorf("read package.json: %w", err)
		}
		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
			PackageManager  string            `json:"packageManager"`
		}
		if err := json.Unmarshal(pkgJSON, &pkg); err == nil {
			frameworkSet := map[string]struct{}{}
			for name := range pkg.Dependencies {
				frameworkSet[name] = struct{}{}
			}
			for name := range pkg.DevDependencies {
				frameworkSet[name] = struct{}{}
			}
			for framework := range frameworkSet {
				switch framework {
				case "next", "nuxt", "@sveltejs/kit", "vite", "astro", "tailwindcss", "react", "vue", "angular", "@angular/core", "strapi":
					ws.Frameworks = append(ws.Frameworks, normalizeFramework(framework))
				}
			}
			if pkg.PackageManager != "" {
				if manager, ok := parsePackageManagerHint(pkg.PackageManager); ok {
					hints[manager] = struct{}{}
				} else {
					hasUnsupportedExplicitManager = true
				}
			}
		}
	}

	for _, hint := range []struct {
		name    string
		manager shared.PackageManager
	}{
		{name: "pnpm-workspace.yaml", manager: shared.PackageManagerPNPM},
		{name: "bunfig.toml", manager: shared.PackageManagerBun},
		{name: "deno.json", manager: shared.PackageManagerDeno},
		{name: "deno.jsonc", manager: shared.PackageManagerDeno},
	} {
		path := filepath.Join(root, hint.name)
		ok, err := d.fs.Stat(ctx, path)
		if err != nil {
			return ws, fmt.Errorf("stat %s: %w", hint.name, err)
		}
		if ok {
			hints[hint.manager] = struct{}{}
		}
	}

	for _, lockfile := range []struct {
		name    string
		manager shared.PackageManager
	}{
		{name: "package-lock.json", manager: shared.PackageManagerNPM},
		{name: "pnpm-lock.yaml", manager: shared.PackageManagerPNPM},
		{name: "bun.lock", manager: shared.PackageManagerBun},
		{name: "bun.lockb", manager: shared.PackageManagerBun},
		{name: "deno.lock", manager: shared.PackageManagerDeno},
	} {
		path := filepath.Join(root, lockfile.name)
		ok, err := d.fs.Stat(ctx, path)
		if err != nil {
			return ws, fmt.Errorf("stat %s: %w", lockfile.name, err)
		}
		if ok {
			ws.Lockfiles = append(ws.Lockfiles, path)
			if ws.PackageManager == shared.PackageManagerUnknown {
				ws.PackageManager = lockfile.manager
			}
		}
	}

	if ws.PackageManager == shared.PackageManagerUnknown {
		switch len(hints) {
		case 1:
			for manager := range hints {
				ws.PackageManager = manager
			}
		case 0:
			if manifestExists && !hasUnsupportedExplicitManager {
				ws.PackageManager = shared.PackageManagerNPM
			}
		}
	}

	sort.Strings(ws.Frameworks)
	sort.Strings(ws.Lockfiles)
	return ws, nil
}

func parsePackageManagerHint(value string) (shared.PackageManager, bool) {
	name := strings.TrimSpace(value)
	if name == "" {
		return shared.PackageManagerUnknown, false
	}
	if idx := strings.Index(name, "@"); idx > 0 {
		name = name[:idx]
	}
	switch strings.ToLower(name) {
	case string(shared.PackageManagerNPM):
		return shared.PackageManagerNPM, true
	case string(shared.PackageManagerPNPM):
		return shared.PackageManagerPNPM, true
	case string(shared.PackageManagerBun):
		return shared.PackageManagerBun, true
	case string(shared.PackageManagerDeno):
		return shared.PackageManagerDeno, true
	default:
		return shared.PackageManagerUnknown, false
	}
}

func normalizeFramework(input string) string {
	input = strings.ToLower(input)
	switch input {
	case "@sveltejs/kit":
		return "sveltekit"
	case "@angular/core":
		return "angular"
	default:
		return strings.TrimPrefix(input, "@")
	}
}
