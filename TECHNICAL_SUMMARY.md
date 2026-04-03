## Celador CLI v1 Technical Summary

### Overview

Celador CLI v1 is implemented as a Go command-line application focused on dependency and supply-chain security for JavaScript and Deno ecosystems. The implementation follows a hexagonal architecture and provides a buildable, testable baseline for workspace hardening, dependency scanning, remediation planning, and guarded install flows.

### Architecture

The repository is organized around a standard Go project layout:

- `cmd/celador` contains the binary entrypoint.
- `internal/app` wires the runtime, dependency graph, and Cobra/Viper commands.
- `internal/core` contains domain models and use cases for workspace management, auditing, remediation, and install preflight checks.
- `internal/ports` defines the interfaces used by the core layer.
- `internal/adapters` implements filesystem, cache, OSV, package-manager, rules, and terminal adapters.
- `configs` contains built-in templates and rule packs.
- `test` contains fixtures, helpers, and end-to-end coverage.

This keeps business logic isolated from external integrations such as OSV.dev, local filesystems, and package-manager execution.

### Delivered Commands

#### `celador init`

- Detects workspace characteristics and package-manager context.
- Applies hardening defaults for supported package-manager configuration files.
- Manages project AI guidance files through preserved managed blocks.
- Generates or updates `AGENTS.md`, `CLAUDE.md`, and `llm.txt` without clobbering unrelated user content.
- Honors non-interactive execution through root runtime settings.

#### `celador scan`

- Parses supported lockfile formats for the current v1 scope.
- Queries OSV.dev through batch requests.
- Loads framework rule packs and produces security findings.
- Supports ignore rules and persistent cache reuse.
- Uses TTL-aware and offline-aware cache behavior to reduce network dependence.

#### `celador fix`

- Produces conservative remediation plans.
- Applies manifest-focused changes intended to minimize project breakage.
- Surfaces fix context and impact boundaries for the current v1 scope.

#### `celador install`

- Provides a scoped zero-trust preflight wrapper.
- Detects selected suspicious install-time patterns before handoff.
- Preserves CI-safe and non-interactive behavior.

### Security and Rules

- OSV batch integration is implemented through a dedicated adapter.
- Built-in rule packs currently include examples such as Next.js and Vite.
- Ignore support is available through managed ignore storage.
- Guidance generation injects supply-chain rules for AI assistants and contributors.

### Caching Strategy

The implementation includes persistent caching designed around:

- lockfile/result reuse,
- TTL-based OSV response reuse,
- offline-aware fallback behavior,
- visible cache-state indicators in scan flows.

### Testing and Validation

The project now includes unit and end-to-end test coverage across core flows and managed-file behavior. Validation reported during SDD execution passed with:

- `go build ./...`
- `go vet ./...`
- `go test ./...`

### Current v1 Boundaries

The implemented v1 intentionally keeps some areas conservative:

- `fix` is primarily manifest-focused rather than fully lockfile-mutating.
- Bun support targets text lockfile handling for the current scope.
- Rich Bubble Tea flows are not yet the dominant interaction layer; terminal behavior exists with TTY-aware fallbacks.
- Some non-critical warnings remain around runtime proof depth and total coverage.

### Important Files

- `go.mod` — module bootstrap and dependencies.
- `cmd/celador/main.go` — CLI entrypoint.
- `internal/app/bootstrap.go` — runtime assembly.
- `internal/app/commands.go` — command tree and flags.
- `internal/core/workspace/service.go` — workspace hardening and managed file updates.
- `internal/core/audit/service.go` — audit orchestration and cache-aware scanning.
- `internal/core/fix/service.go` — remediation planning and application.
- `internal/core/install/service.go` — install preflight workflow.
- `internal/adapters/osv/client.go` — OSV batch client.
- `internal/adapters/cache/file_cache.go` — persistent cache implementation.
- `internal/adapters/fs/templates.go` — managed template rendering and merge preservation.

### Final Status

The SDD change `implement-celador-cli-v1` was archived as pass-with-warnings: the core CLI v1 scope is delivered and validated, with follow-up improvements recommended for broader fix application, deeper proof of some runtime behaviors, and higher overall coverage.
