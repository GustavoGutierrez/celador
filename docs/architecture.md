# Overview and Architecture

Celador is a Go CLI for dependency and supply-chain hardening in JavaScript, TypeScript, and Deno workspaces.

The current implementation focuses on four user workflows:

- `celador init` bootstraps workspace hardening and managed guidance files.
- `celador scan` audits supported lockfiles and applies built-in framework/security rules.
- `celador fix` plans and applies conservative manifest-level remediation.
- `celador install` performs a preflight risk assessment before delegating installation to the workspace package manager.

## Architecture

Celador follows a hexagonal architecture.

- `cmd/celador` — binary entrypoint
- `internal/app` — runtime bootstrap and Cobra command wiring
- `internal/core` — domain services for workspace hardening, auditing, remediation, and install preflight
- `internal/ports` — interfaces consumed by the core layer
- `internal/adapters` — filesystem, cache, OSV, package manager, rules, and terminal adapters
- `configs` — built-in rule packs and managed templates
- `test` — fixtures, helpers, and end-to-end coverage

## Runtime model

At startup, Celador:

1. Uses the current working directory as the workspace root.
2. Loads optional configuration from `.celador.yaml` in that root.
3. Creates a persistent cache under `.celador/cache`.
4. Detects whether the process is running in a TTY and whether `CI` is set.
5. Wires the services used by `init`, `scan`, `fix`, and `install`.

## Current workspace detection

Celador currently detects:

- package managers from lockfiles: `package-lock.json`, `pnpm-lock.yaml`, `bun.lock`, `bun.lockb`, `deno.lock`
- frameworks from `package.json` dependency names: `next`, `nuxt`, `@sveltejs/kit`, `vite`, `astro`, `tailwindcss`, `react`, `vue`, `angular`, `@angular/core`, `strapi`

Framework detection is broader than the current rule packs. See [Framework and security rules](security-rules.md) for the active enforcement scope.
