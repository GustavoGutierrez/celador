# Contributing to Celador

Thanks for contributing. Celador enforces a strict **issue-first workflow** — every change starts with an approved issue, no exceptions.

---

## Contribution Workflow

```
Open Issue → Get status:approved → Fork & Branch → Implement → Open PR → CI → Review & Merge
```

### Step 1: Open an Issue

Use the correct template:

- **Bug Report** — reproducible defect in existing behavior
- **Feature Request** — new capability or improvement
- **Security Vulnerability** — use the private channel described in [SECURITY.md](SECURITY.md), not a public issue

Blank issues are not accepted. Fill in all required fields. New issues receive the `status:needs-review` label automatically.

### Step 2: Wait for Approval

A maintainer will review the issue and add `status:approved` if it is accepted for implementation.

**Do not open a PR until the issue is approved.** PRs that reference unapproved issues will not be reviewed.

### Step 3: Fork, Branch, and Implement

Once the issue is approved:

1. Fork the repository and create a branch from `main`
2. Name your branch after the issue and type: `feat/123-sarif-flag`, `fix/456-cache-invalidation`
3. Implement your change following the coding conventions below
4. Run the full test suite locally before pushing (see [Testing](#testing))

### Step 4: Open a Pull Request

1. Open a PR targeting `main` — link the approved issue with `Closes #N`
2. Use the PR template and fill in all sections
3. Add exactly **one `type:*` label** to the PR

### Step 5: Automated CI Checks

Every PR must pass these checks before it can be merged:

| Check | Command |
|---|---|
| Unit tests | `go test ./...` |
| Build | `go build ./...` |
| Vet | `go vet ./...` |
| PR has issue reference | Body contains `Closes #N`, `Fixes #N`, or `Resolves #N` |
| PR has `type:*` label | Exactly one type label present |

All checks must be green. Fix failures before requesting review.

---

## Testing

Run the full suite locally before every push:

```bash
go test ./...
go build ./...
go vet ./...
```

For release-related changes, also validate the GoReleaser config and Homebrew formula:

```bash
go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml
ruby -c packaging/homebrew/Formula/celador.rb
```

**Coverage targets:**

- Security-critical paths (OSV client, patch writer, workspace service): 80%+
- New adapters and core services: tests required before merging
- New CLI commands: at least one integration test via `commands_test.go`

---

## Architecture

Celador uses **hexagonal architecture (ports and adapters)**. Keep this separation clean:

| Layer | Path | Rule |
|---|---|---|
| CLI wiring | `internal/app/` | Cobra commands and bootstrap only — no business logic |
| Domain services | `internal/core/` | All business logic lives here |
| Interfaces | `internal/ports/` | Define boundaries between core and adapters |
| Implementations | `internal/adapters/` | One adapter per external concern |
| Binary entrypoint | `cmd/celador/` | Calls `app.Bootstrap` and exits |

**Key rules:**
- Define new integration boundaries in `internal/ports/` before writing adapter implementations
- Keep business logic in `internal/core/` — never push domain rules into `internal/app/` or adapters
- Wire new adapters in `internal/app/bootstrap.go` — that is the single composition root
- Keep Cobra-specific concerns in `internal/app/commands.go`

---

## Coding Conventions

### General

- Follow idiomatic Go
- Prefer small interfaces and explicit constructor-based wiring
- Use early returns and clear error wrapping with `fmt.Errorf("context: %w", err)`
- Keep package names short and intention-revealing
- Match existing output style before changing CLI behavior

### Security rules (mandatory)

These rules are enforced on all new code:

**1. Never discard errors silently**

```go
// Bad
value, _ := someFunction()

// Good — distinguish missing from failed
content, err := s.fs.ReadFile(ctx, path)
if err != nil {
    if os.IsNotExist(err) {
        content = []byte{}
    } else {
        return fmt.Errorf("read %s: %w", path, err)
    }
}
```

**2. Validate all CLI input**

```go
for _, arg := range args {
    if strings.TrimSpace(arg) == "" {
        return NewExitError(2, "package name cannot be empty or whitespace")
    }
}
```

**3. Prevent path traversal**

```go
absPath, err := filepath.Abs(path)
if err != nil {
    return err
}
if !strings.HasPrefix(absPath, ws.Root) {
    return fmt.Errorf("path %q is outside workspace", path)
}
```

**4. Guard against empty external API calls**

```go
if len(deps) == 0 {
    return []shared.Finding{}, nil
}
```

**5. External endpoints must be configurable**

```go
endpoint := os.Getenv("CELADOR_OSV_ENDPOINT")
if endpoint == "" {
    endpoint = "https://api.osv.dev/v1/querybatch"
}
```

---

## Conventional Commits

All commits must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short description>

[optional body]

[optional footer]
```

**Examples:**

```
feat(scan): add --sarif flag for SARIF v2.1.0 output

fix(cache): invalidate entries when schema version changes

docs(readme): update supported versions table

refactor(audit): extract typosquatting detection into service

chore(deps): bump github.com/dop251/goja to v0.0.0-20250309

fix!: change cache key format (breaking change)
BREAKING CHANGE: existing .celador/cache entries are invalidated on upgrade
```

Types and their corresponding PR labels:

| Commit type | PR label |
|---|---|
| `feat` | `type:feature` |
| `fix` | `type:bug` |
| `docs` | `type:docs` |
| `refactor` | `type:refactor` |
| `chore` | `type:chore` |
| `fix!` / `BREAKING CHANGE` | `type:breaking-change` |

**Never add `Co-authored-by:` trailers.** All commits must have only the human author.

---

## Label System

### Type labels (required on every PR — pick exactly one)

| Label | Use for |
|---|---|
| `type:bug` | Bug fixes |
| `type:feature` | New features |
| `type:docs` | Documentation-only changes |
| `type:refactor` | Code refactoring with no behavior change |
| `type:chore` | Maintenance, tooling, dependency updates |
| `type:breaking-change` | Breaking changes (requires major version bump) |

### Status labels (set by maintainers)

| Label | Meaning |
|---|---|
| `status:needs-review` | Awaiting maintainer review (auto-applied to new issues) |
| `status:approved` | Approved — PRs can now be opened |
| `status:in-progress` | Actively being worked on |
| `status:blocked` | Blocked by another issue or external dependency |
| `status:stale` | No activity for 30 days |
| `status:wontfix` | Intentionally not fixing |

### Priority labels (set by maintainers)

`priority:high`, `priority:medium`, `priority:low`

### Effort labels (set by maintainers)

| Label | Meaning |
|---|---|
| `effort:small` | < 1 hour — good first issue |
| `effort:medium` | 1–4 hours |
| `effort:large` | > 4 hours or spans multiple files |

---

## PR Rules

- One logical change per PR — keep scope focused
- Update documentation in the same PR when behavior changes
- Do not modify unrelated files
- Do not change release automation or Homebrew packaging without discussing it in an issue first
- Ensure `go test ./...` passes before requesting review
- Do not include `Co-Authored-By` trailers in commits

---

## Maintainer Triage Cadence

| Activity | Frequency | What Happens |
|---|---|---|
| New issue triage | Within 3 days | Labeled + approved or closed with explanation |
| PR review | Within 7 days | Review started; changes requested or merged |
| Stale sweep | Monthly | Issues and PRs with no activity flagged |

If you have not received a response within 7 days on a PR or issue, a single follow-up comment is welcome.

---

## What Gets Closed Without Merging

- PRs opened without an approved issue
- PRs that fail CI and are not updated within 14 days
- Issues that are vague, duplicates, or lack reproduction steps after a maintainer request
- Issues with no response to a maintainer question within 14 days

---

## Questions and Discussions

For questions, ideas, or informal discussion that are not actionable bugs or feature requests, open a [GitHub Discussion](https://github.com/GustavoGutierrez/celador/discussions) instead of an issue.
