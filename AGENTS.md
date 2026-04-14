# AGENTS.md

Practical guidance for AI agents and contributors working in the Celador repository.

## Project rules

- Keep all code, documentation, comments, and contributor-facing instructions in English.
- Do not modify unrelated files when completing a focused task.
- Prefer small, targeted changes that preserve the current architecture.
- Follow existing repository structure and naming before introducing new patterns.
- Treat release automation and Homebrew publishing as production-critical workflows.
- **NEVER add `Co-authored-by:` trailers to commit messages.** All commits must have only the human author's sign-off. No AI or automated contributor attribution in commit metadata.

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

## Security and error handling rules (MANDATORY)

These rules address identified gaps from project analysis and must be followed for all new code:

### 1. NEVER discard errors silently

- All errors must be handled, wrapped with context, or explicitly documented if intentionally ignored.
- **Never use** `value, _ := function()` pattern unless the error truly has zero impact and is justified in a comment.
- When reading files where absence is acceptable (e.g., optional configs), distinguish between "file does not exist" and "read failed":

  ```go
  content, err := s.fs.ReadFile(ctx, path)
  if err != nil {
      if os.IsNotExist(err) {
          content = []byte{} // Safe: file doesn't exist, will create
      } else {
          return fmt.Errorf("read %s: %w", path, err)
      }
  }
  ```

### 2. Validate all CLI input

- CLI arguments must be validated for empty strings, whitespace-only values, and malicious patterns:

  ```go
  for _, arg := range args {
      if strings.TrimSpace(arg) == "" {
          return NewExitError(2, "package name cannot be empty or whitespace")
      }
  }
  ```

### 3. Prevent path traversal

- When accepting file paths from user config or CLI, validate they remain within the workspace root:

  ```go
  absPath, err := filepath.Abs(path)
  if err != nil {
      return err
  }
  if !strings.HasPrefix(absPath, ws.Root) {
      return fmt.Errorf("path %q is outside workspace", path)
  }
  ```

### 4. Preserve severity data from external APIs

- Never hard-code severity levels when the upstream API provides them (e.g., OSV vulnerabilities).
- Parse and propagate actual severity; use reasonable defaults only when data is missing.

### 5. Guard against empty external API calls

- Skip network calls when there is nothing to query:

  ```go
  if len(deps) == 0 {
      return []shared.Finding{}, nil
  }
  ```

### 6. Use public port interfaces consistently

- Always use `ports.Clock`, `ports.Logger`, and other public port interfaces.
- Never redefine private duplicates of existing port interfaces.

### 7. External endpoints must be configurable

- External API endpoints should support environment variable overrides:

  ```go
  endpoint := os.Getenv("CELADOR_OSV_ENDPOINT")
  if endpoint == "" {
      endpoint = "https://api.osv.dev/v1/querybatch"
  }
  ```

### 8. Handle binary formats correctly

- When claiming support for binary file formats (e.g., `bun.lockb`), implement proper parsing or explicitly decline support with a user warning.

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

- **Target 80%+ test coverage** for security-critical paths (OSV client, patch writer, workspace service).
- New adapters and core services must include tests before merging.

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
- `celador-security`
  - Use when implementing security-critical features, handling user input, writing file operations,
    integrating external APIs, or reviewing code for security vulnerabilities and error handling gaps.

## Practical workflow notes

- Read the relevant docs before changing release behavior:
  - `README.md`
  - `packaging/homebrew/README.md`
  - `packaging/homebrew/RELEASE_PROCESS.md`
  - `.agents/skills/celador-release/SKILL.md`
- If a task changes installation or release behavior, keep README, release docs, and skill guidance in
  sync.
- If a change affects user-facing commands, document the exact commands users should run.
- When implementing new features, invoke the `celador-security` skill to ensure compliance with
  error handling, input validation, and security rules documented above.
