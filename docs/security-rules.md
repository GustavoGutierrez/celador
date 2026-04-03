# Framework and Security Rules

## Scope note

Celador can detect more frameworks than it currently enforces with built-in rules. The current implemented rule scope is intentionally smaller than the framework detection list.

## Current finding sources

Celador currently reports findings from:

- OSV vulnerability lookups
- built-in framework/file rules from `configs/rules/*.yaml`
- a Tailwind arbitrary-value heuristic scan
- install preflight heuristics in `celador install`

## Built-in rule packs in the current repository

### Next.js

Source: `configs/rules/next.yaml`

Current checks:

- `next-powered-by-header`
  - file: `next.config.js`
  - requires `poweredByHeader: false`
- `next-remote-pattern-wildcard`
  - file: `next.config.js`
  - flags `hostname: '*'`

### Vite

Source: `configs/rules/vite.yaml`

Current check:

- `vite-production-sourcemap`
  - file: `vite.config.ts`
  - flags `sourcemap: true`

The Vite rule is evaluated for workspaces detected as `vite`, `react`, or `vue`.

## Tailwind heuristic

Celador also scans `.tsx`, `.vue`, and `.svelte` files for a simple Tailwind arbitrary-value pattern.

Current heuristic:

- finding ID: `tailwind-dynamic-arbitrary-value`
- flags files containing both `bg-[` and `+`

This is a conservative string heuristic, not a full parser.

## Framework detection versus active enforcement

The detector currently recognizes these framework names from `package.json` dependencies:

- `next`
- `nuxt`
- `sveltekit`
- `vite`
- `astro`
- `tailwindcss`
- `react`
- `vue`
- `angular`
- `strapi`

However, the current built-in file rule packs only cover:

- Next.js
- Vite-family projects

Plus the Tailwind heuristic described above.

## Ignore interactions

You can suppress findings temporarily with `.celadorignore` by matching either:

- the finding ID
- the finding target path

This applies to both OSV and rule-generated findings when the selector matches.
