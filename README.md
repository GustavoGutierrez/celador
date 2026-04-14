<p align="center">
  <img src="celador.png" width="500" alt="Celador CLI" />
</p>

# Celador CLI

> Supply chain security for JavaScript, TypeScript, and Deno workspaces.

Celador is a Go CLI that audits npm/pnpm/Bun/Deno dependencies for known vulnerabilities, flags risky framework configuration, inspects package tarballs at install time, and applies conservative manifest-level remediations. It is built for deterministic, non-interactive use in both developer machines and CI/CD pipelines.

**Current version:** v0.6.0 — [see releases](https://github.com/GustavoGutierrez/celador/releases)

---

## What Celador actually does

These are the capabilities that are implemented, tested, and wired into the CLI today:

| Capability | Command | How it works |
|---|---|---|
| **Known vulnerability scanning** | `scan` | Queries OSV.dev in parallel batches; results cached for 24h with offline fallback |
| **Framework configuration rules** | `scan` | YAML rule packs detect insecure framework settings (e.g. `sourcemap: true`, `poweredByHeader`) |
| **Tarball content inspection** | `install <pkg>` | Downloads the package tarball and scans source files for `eval()`, `child_process.exec/spawn`, `process.env` + network co-occurrence, hex-encoded strings, undocumented `.node` binaries, and lifecycle scripts (`preinstall`, `install`, `postinstall`, `prepare`, `prepublish`) |
| **Conservative remediation** | `fix` | Plans and applies semver-safe `package.json` updates — skips major bumps and prerelease targets |
| **Workspace hardening** | `init` | Writes `.celador.yaml`, `.celadorignore`, updates `.gitignore`/`.npmignore`, hardens package manager config, and writes managed guidance to `AGENTS.md` |
| **Persistent cache** | `scan`, `fix` | Fingerprint-keyed scan and OSV results; automatically invalidated when lockfile or rules change |
| **CI/CD-friendly output** | `scan` | `--json` flag for structured output; exit codes `0` (clean), `3` (findings), `4` (no safe fix) |
| **Offline fallback** | `scan` | Uses stale cached OSV results when the API is unreachable |
| **Version update checks** | `--version` | Async GitHub release check with Homebrew upgrade guidance |

> **Important:** The tarball inspection (eval, exec, lifecycle scripts, hex obfuscation, `.node` binaries) runs **only during `celador install`**, not during `celador scan`. The `scan` command audits lockfile dependencies against OSV and YAML rules only.

---

## What Celador does NOT do (yet)

The following capabilities were designed and their adapters were written, but are **not yet wired into any CLI command**. They compile and have tests, but produce no output when you run the CLI today:

| Feature | Status | PRD |
|---|---|---|
| **SARIF output** (`celador scan --sarif`) | Adapter implemented, not wired | PRD-004 |
| **SBOM / SPDX generation** (`celador scan --sbom`) | Adapter implemented, not wired | PRD-005 |
| **Cryptographic provenance** (`celador install --verify-provenance`) | Adapter implemented, not wired | PRD-007 |
| **Behavioral sandbox** (`celador install --sandbox-scan`) | Adapter implemented (goja engine), not wired | PRD-008 |
| **Typosquatting detection during scan** | `DetectTyposquat()` implemented, not called from audit pipeline | PRD-003 |

These are the next natural integration steps. The logic exists — what remains is wiring each adapter into the CLI, adding flags, and integrating into the relevant command flows.

### Scope limitations (by design)

- **JS/TS/Deno ecosystems only.** No support for Python, Rust, Java, Go, or other ecosystems.
- **No monorepo support.** Celador operates on a single workspace root directory.
- **`celador fix` modifies `package.json` only.** It does not run the package manager after patching or update lockfiles directly.
- **`celador fix` re-runs the full scan pipeline.** It does not reuse a previous `scan` result; this doubles scan time when both commands are used in sequence.
- **No automated pull requests.** Remediations are applied locally; there is no GitHub/GitLab PR automation.
- **Tarball inspection is heuristic, not exhaustive.** Static pattern matching catches common patterns (eval, exec, network + env co-occurrence) but not multi-layer obfuscation chains (e.g. `eval(atob(...))`), protestware with conditional logic, or malicious behavior that only activates at runtime.
- **Cache is per full scan fingerprint, not per package.** Adding one new dependency invalidates and rescans all existing ones.

---

## Installation

### macOS and Linux (Homebrew)

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
```

### Windows

Download the release asset from GitHub Releases:

- <https://github.com/GustavoGutierrez/celador/releases>
- `celador_X.Y.Z_windows_amd64.zip`

---

## Updating

### Homebrew (macOS and Linux)

```bash
brew update && brew upgrade GustavoGutierrez/celador/celador
```

### Windows

Download the latest release asset and replace the existing binary.

---

## Quick usage

```bash
celador --version          # print version; check for newer release
celador init               # bootstrap workspace hardening
celador scan               # audit dependencies and config rules
celador scan --json        # structured JSON output for CI tooling
celador fix --diff         # preview planned remediations
celador fix --yes          # apply remediations without prompting
celador install express    # preflight-check a package before installing
celador tui                # interactive overview (TTY) or static fallback
celador about              # project and release info
```

---

## Environment variables

| Variable | Purpose |
|---|---|
| `CELADOR_OSV_ENDPOINT` | Override OSV batch query endpoint (default: `https://api.osv.dev/v1/querybatch`) |
| `CELADOR_OSV_VULN_API` | Override individual vulnerability details endpoint |
| `CELADOR_NPM_REGISTRY` | Override npm registry for provenance checks (adapter only, not yet wired) |

These are useful for corporate proxies, air-gapped environments, or local OSV mirrors. Celador behaves identically to previous versions when none are set.

---

## CI/CD integration

```bash
# Fail the pipeline on any findings
celador scan --no-interactive
# exit 0 = clean
# exit 3 = findings present

# Apply safe remediations unattended
celador fix --yes --no-interactive
```

---

## Architecture and contributing

Celador uses **hexagonal architecture (ports and adapters)** written in Go.

- `cmd/celador` — binary entrypoint
- `internal/app` — Cobra command tree and runtime wiring
- `internal/core` — domain services and use cases
- `internal/ports` — interfaces between core and adapters
- `internal/adapters` — filesystem, cache, OSV, package managers, output formats, sandbox, provenance, rules, and terminal I/O
- `configs/rules` — bundled YAML security rule packs

Contributor guidance and project rules live in [`AGENTS.md`](AGENTS.md).
Architecture details live in [`docs/architecture.md`](docs/architecture.md).

---

## Documentation

- [`docs/commands.md`](docs/commands.md) — full command reference
- [`docs/configuration.md`](docs/configuration.md) — `.celador.yaml`, cache layout, and environment variables
- [`docs/security-rules.md`](docs/security-rules.md) — built-in YAML rule packs
- [`docs/installation.md`](docs/installation.md) — install and update instructions
- [`docs/architecture.md`](docs/architecture.md) — hexagonal architecture overview
- [`docs/FORENSIC_ANALYSIS.md`](docs/FORENSIC_ANALYSIS.md) — honest assessment of detection coverage and known gaps (as of v0.4.5; PRD-007 and PRD-008 adapters added since)

---

## License

MIT License — Copyright (c) 2026 Gustavo Gutierrez. See [LICENSE](LICENSE) for the full text.
