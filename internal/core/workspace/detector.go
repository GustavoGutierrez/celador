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
	if ok, err := d.fs.Stat(ctx, manifest); err != nil {
		return ws, fmt.Errorf("stat package.json: %w", err)
	} else if ok {
		ws.ManifestPath = manifest
		pkgJSON, err := d.fs.ReadFile(ctx, manifest)
		if err != nil {
			return ws, fmt.Errorf("read package.json: %w", err)
		}
		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
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

	sort.Strings(ws.Frameworks)
	sort.Strings(ws.Lockfiles)
	return ws, nil
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
