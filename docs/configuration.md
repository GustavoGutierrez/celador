# Configuration, Cache, Ignores, and Managed Files

## `.celador.yaml`

Celador reads `.celador.yaml` from the workspace root.

`celador init` ensures these defaults exist:

```yaml
cache:
  ttl: 24h
rules:
  version: v1
output:
  plain_text: true
```

Current runtime usage:

- `cache.ttl` controls OSV cache freshness

Current notes:

- `rules.version` is written by `init` as metadata for the rule pack version Celador expects
- `output.plain_text` documents the current plain-text runtime surface

## Cache layout

Celador stores cache files under:

```text
.celador/cache/
```

Current cache types:

- `scan-<fingerprint>.json` — cached full scan results
- `osv-<fingerprint>.json` — cached OSV findings with expiration metadata

The cache fingerprint includes:

- lockfile content
- ignore rules
- loaded rule version

## Ignore behavior

Ignore rules are loaded from `.celadorignore`.

Default header:

```text
# finding-id|reason|expires-at(YYYY-MM-DD)
```

Rule format:

```text
selector|reason|expires-at
```

Examples:

```text
GHSA-xxxx-temporary|accepted until upstream release|2026-05-01
next.config.js|legacy exception for migration|2026-06-15
```

Current matching behavior:

- first tries the finding ID
- then tries the finding target
- ignores only apply while the expiration date is empty or still in the future

## Files managed by `celador init`

### Hardening and project files

- `.celador.yaml`
- `.celadorignore`
- `.gitignore`
- `.npmignore`
- `.npmrc` for npm and pnpm workspaces
- `bunfig.toml` for Bun workspaces
- `deno.json` for Deno workspaces

### Guidance files

- `AGENTS.md`
- `CLAUDE.md` only when it already exists

### Optional hook

- `.git/hooks/pre-commit` when `--install-hook` is used

## Current hardening changes applied by `init`

### Ignore hygiene

Celador appends these entries to `.gitignore` and `.npmignore` if they are missing, preserving any existing content already in those files:

- `.env.local`
- `*.map.js`
- `*.js.map`
- `coverage/`

Celador also appends this cache entry to `.gitignore` when it is missing:

- `.celador/`

### npm and pnpm workspaces

Celador writes or updates `.npmrc` with:

```ini
ignore-scripts=true
minimum-release-age=1440
save-exact=true
trust-policy=no-downgrade
```

### Bun workspaces

Celador writes or updates `bunfig.toml` with:

- `install.saveExact = true`
- `install.minimumReleaseAge = 1440`
- `install.minimumReleaseAgeExcludes = ["webpack", "react", "typescript", "vite", "next", "nuxt"]`

### Deno workspaces

Celador writes or updates `deno.json` with:

- `lock = true`

## Managed block behavior

For `AGENTS.md` and `CLAUDE.md`, Celador manages only the section between:

```html
<!-- celador:start -->
<!-- celador:end -->
```

Content outside those markers is preserved.

`celador init` always creates or updates `AGENTS.md`. It only touches `CLAUDE.md` when that file is already present in the target workspace. It does not create `llm.txt` in target workspaces.
