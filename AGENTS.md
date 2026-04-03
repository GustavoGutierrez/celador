# AGENTS.md

Practical guidance for AI agents and contributors working in the Celador repository.

## Project rules

- Keep all code, documentation, comments, and contributor-facing instructions in English.
- Do not modify unrelated files when completing a focused task.
- Prefer small, targeted changes that preserve the current architecture.
- Follow existing repository structure and naming before introducing new patterns.
- Treat release automation and Homebrew publishing as production-critical workflows.

## Architecture

Celador is a Go CLI built around hexagonal architecture (ports and adapters).

- `cmd/celador` — binary entrypoint
- `internal/app` — runtime wiring, bootstrap, Cobra command tree
- `internal/core` — domain services and use cases
- `internal/ports` — interfaces consumed by the core layer
- `internal/adapters` — implementations for filesystem, cache, OSV, package managers, rules, and terminal I/O
- `configs` — bundled templates and rules
- `test` — fixtures, helpers, and end-to-end coverage

## Design decisions

- Keep business logic in `internal/core`; avoid pushing domain rules into CLI wiring or adapters.
- Define new integration boundaries in `internal/ports` before adding adapter implementations.
- Keep Cobra-specific concerns in `internal/app` unless a reusable domain abstraction is required.
- Preserve deterministic, non-interactive CLI behavior by default.
- Maintain managed-file behavior carefully when updating generated guidance or template content.

## Release and Homebrew decisions

- GitHub releases are published from `.github/workflows/release.yml` using `.goreleaser.yaml`.
- Celador release tags must follow Semantic Versioning in the form `vMAJOR.MINOR.PATCH`.
- Homebrew publishing uses the dedicated tap repository `GustavoGutierrez/homebrew-celador`.
- User-facing Homebrew install commands are:

  ```bash
  brew tap GustavoGutierrez/celador
  brew install GustavoGutierrez/celador/celador
  ```

- Homebrew resolves `brew tap GustavoGutierrez/celador` to `GustavoGutierrez/homebrew-celador`.
- Windows is not installed through Homebrew; users must download the GitHub release asset instead.
- Tap publishing requires the `HOMEBREW_TAP_SSH_KEY` secret in the source repository. It must contain
  the private half of a write-enabled deploy key registered on `GustavoGutierrez/homebrew-celador`.

## Coding conventions

- Follow idiomatic Go.
- Prefer small interfaces, explicit dependencies, and constructor-based wiring.
- Use early returns and clear error wrapping.
- Keep package names short and intention-revealing.
- Match the existing output style and command ergonomics before changing CLI behavior.
- For docs, prefer practical instructions, exact commands, and operational caveats.

## Testing expectations

- At minimum, run `go test ./...` for Go code changes.
- When touching runtime wiring, CLI behavior, or release-related logic, also consider `go build ./...`
  and `go vet ./...`.
- For release workflow changes, validate the Homebrew formula path and GoReleaser config:

  ```bash
  go test ./...
  go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml
  ruby -c packaging/homebrew/Formula/celador.rb
  ```

## Local skills available in this project

Use these local skills when the task matches their scope:

- `golang-patterns`
  - Use for idiomatic Go structure, error handling, interfaces, naming, and maintainability decisions.
- `golang-pro`
  - Use for advanced Go work such as concurrency, performance-sensitive flows, service boundaries,
    benchmarks, or sophisticated testing patterns.
- `celador-release`
  - Use when publishing a release, re-running a release for an existing tag, validating release
    assets, or verifying/publishing the dedicated Homebrew tap.

## Practical workflow notes

- Read the relevant docs before changing release behavior:
  - `README.md`
  - `packaging/homebrew/README.md`
  - `packaging/homebrew/RELEASE_PROCESS.md`
  - `.agents/skills/celador-release/SKILL.md`
- If a task changes installation or release behavior, keep README, release docs, and skill guidance in
  sync.
- If a change affects user-facing commands, document the exact commands users should run.
