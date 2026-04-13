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

## Environment variables for enterprise and proxy configurations

Celador supports environment variables to override default external API endpoints. These are **optional** — if not set, Celador uses its built-in defaults and behaves identically to previous versions.

### OSV API endpoints

| Variable | Default | Purpose |
|----------|---------|---------|
| `CELADOR_OSV_ENDPOINT` | `https://api.osv.dev/v1/querybatch` | Batch vulnerability query endpoint |
| `CELADOR_OSV_VULN_API` | `https://api.osv.dev/v1/vulns` | Individual vulnerability details endpoint |

### Usage examples

**Default behavior (no configuration needed):**

```bash
celador scan
# Uses https://api.osv.dev endpoints automatically
```

**Corporate proxy (one-time override):**

```bash
CELADOR_OSV_ENDPOINT=https://proxy.company.com/osv/querybatch \
CELADOR_OSV_VULN_API=https://proxy.company.com/osv/vulns \
celador scan
```

**Persistent configuration (entire terminal session):**

```bash
export CELADOR_OSV_ENDPOINT=https://proxy.company.com/osv/querybatch
export CELADOR_OSV_VULN_API=https://proxy.company.com/osv/vulns

celador scan
celador install express
celador fix --diff
```

**Air-gapped environments (self-hosted OSV instance):**

```bash
export CELADOR_OSV_ENDPOINT=https://internal-osv.company.local/v1/querybatch
export CELADOR_OSV_VULN_API=https://internal-osv.company.local/v1/vulns

celador scan
```

### When to use these variables

- **Corporate network with proxy**: Redirect OSV API calls through your organization's proxy server
- **Air-gapped environments**: Use a self-hosted OSV instance that mirrors the public OSV database
- **Testing/development**: Point to a mock OSV server for integration testing
- **Rate limiting compliance**: Route through a cached internal endpoint to reduce external API calls

### Important notes

- These variables only affect OSV API endpoints. Other external services (npm registry, GitHub releases) use their own defaults.
- Values must be full URLs including the protocol (`https://`).
- If an endpoint is unreachable, Celador will return an error rather than silently falling back to defaults.

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
