## Celador CLI Technical Summary

### Overview

Celador is a Go command-line application focused on dependency and supply-chain security for JavaScript, TypeScript, and Deno workspaces. The implementation follows a hexagonal architecture and provides a buildable, testable baseline for workspace hardening, dependency scanning, remediation planning, and guarded install flows.

### Architecture

The repository is organized around a standard Go project layout:

- `cmd/celador` contains the binary entrypoint.
- `internal/app` wires the runtime, dependency graph, and Cobra commands.
- `internal/core` contains domain models and use cases for workspace management, auditing, remediation, and install preflight checks.
- `internal/ports` defines the interfaces used by the core layer.
- `internal/adapters` implements filesystem, cache, OSV, package-manager, rules, and terminal adapters.
- `configs` contains built-in rule packs and managed template content.
- `test` contains fixtures, helpers, and end-to-end coverage.

### Delivered Commands

#### `celador init`

- Detects workspace characteristics and package-manager context.
- Applies hardening defaults for supported package-manager configuration files.
- Manages project AI guidance files through preserved managed blocks.
- Creates or updates `AGENTS.md`, updates `CLAUDE.md` only when it already exists, and preserves unrelated user content.
- Can install a pre-commit hook when `--install-hook` is explicitly requested.

#### `celador scan`

- Parses supported lockfile formats in the current scope.
- Queries OSV.dev through batch requests with cache support.
- Loads framework rule packs and produces security findings.
- Supports ignore rules and persistent cache reuse.
- Uses TTL-aware and offline-aware cache behavior to reduce network dependence.

#### `celador fix`

- Produces conservative remediation plans.
- Applies manifest-focused changes to `package.json`.
- Bumps direct dependency versions when possible and otherwise writes overrides.
- Does not currently update lockfiles directly.

#### `celador install`

- Provides a scoped zero-trust preflight wrapper.
- Assesses the first requested package before installation.
- Detects suspicious install-time patterns before handing off to npm, pnpm, or Bun.
- Preserves CI-safe and non-interactive behavior.

### Security and Rules

- OSV integration is implemented through a dedicated adapter.
- Built-in rule packs currently cover Next.js and Vite-family configuration checks.
- Tailwind arbitrary values are checked with a simple string heuristic.
- Ignore support is available through managed ignore storage.
- Guidance generation injects supply-chain rules for AI assistants and contributors.

### Caching Strategy

The implementation includes persistent caching designed around:

- lockfile/result reuse,
- TTL-based OSV response reuse,
- offline-aware fallback behavior,
- visible cache-state indicators in scan flows.

### Testing and Validation

The project includes unit and end-to-end test coverage across core flows and managed-file behavior. Common validation commands are:

- `go build ./...`
- `go vet ./...`
- `go test ./...`

### Current Boundaries

The current implementation intentionally keeps some areas conservative:

- `fix` is manifest-focused rather than fully lockfile-mutating.
- install handoff is supported for npm, pnpm, and Bun, but not Deno.
- framework detection is broader than the active built-in rule packs.
- terminal behavior is plain-text first, with prompting only when a TTY is available.

### Important Files

- `go.mod` — module bootstrap and dependencies.
- `cmd/celador/main.go` — CLI entrypoint.
- `internal/app/bootstrap.go` — runtime assembly.
- `internal/app/commands.go` — command tree and flags.
- `internal/core/workspace/service.go` — workspace hardening and managed file updates.
- `internal/core/audit/service.go` — audit orchestration and cache-aware scanning.
- `internal/core/fix/service.go` — remediation planning and application.
- `internal/core/install/service.go` — install preflight workflow.
- `internal/adapters/osv/client.go` — OSV client.
- `internal/adapters/cache/file_cache.go` — persistent cache implementation.
- `internal/adapters/fs/templates.go` — managed section rendering and merge preservation.

### Status

The repository currently delivers the documented CLI scope with release automation for GitHub Releases and a dedicated Homebrew tap, plus Windows release-asset distribution through GitHub Releases.
