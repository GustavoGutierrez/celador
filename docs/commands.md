# Command Reference

## Root command

```text
celador
```

Global flag:

- `--no-interactive` — disables prompts and TTY flows

Available commands:

- `init`
- `scan`
- `fix`
- `install`
- `completion`

## `celador init`

Bootstrap Celador hardening for the current workspace.

```bash
celador init
celador init --install-hook
celador --no-interactive init
```

Flags:

- `--install-hook` — installs `.git/hooks/pre-commit` with `celador scan`

Behavior:

- detects the workspace package manager and frameworks
- writes or merges `.celador.yaml`
- ensures `.celadorignore` exists
- updates `.gitignore` and `.npmignore`
- applies package-manager hardening config
- updates managed guidance in `AGENTS.md`, `CLAUDE.md`, and `llm.txt`
- validates strict `package.json` `engines`

Important constraints:

- does not accept positional arguments
- fails if no supported workspace lockfile is found
- fails if `package.json` has no `engines` field or if an engine version uses `^`, `~`, `>`, or `<`

## `celador scan`

Audit supported lockfiles and active framework rules.

```bash
celador scan
celador --no-interactive scan
```

Behavior:

- requires at least one supported lockfile
- parses dependencies from the detected lockfile set
- loads ignore rules from `.celadorignore`
- loads built-in YAML rule packs from `configs/rules`
- queries OSV.dev with cache reuse and offline fallback behavior
- renders plain-text findings

Exit behavior:

- exit code `0` when no findings remain after ignores
- exit code `3` when findings are detected
- exit code `2` for invalid CLI usage

Output behavior:

- prints a scan fingerprint
- prints the finding count and ignored finding count
- prints one line per finding with the finding ID, package or target context, summary text, and fix version when available
- prints `Result source: cache` when OSV data came from cache
- prints `Mode: offline fallback` when a stale cached result is used because OSV could not be reached

## `celador fix`

Plan or apply conservative remediation.

```bash
celador fix --diff
celador fix --yes
celador --no-interactive fix --yes
```

Flags:

- `--diff` — show the planned diff only
- `--yes` — apply without prompting

Behavior:

- runs the same scan pipeline used by `celador scan`
- builds manifest-level operations from fixable findings
- updates `package.json` only
- bumps existing dependencies when possible
- otherwise writes package overrides

Current scope:

- does not update lockfiles directly
- does not run a package-manager install after writing `package.json`
- returns a dry-run diff preview for the plan

Exit behavior:

- exit code `4` when no safe remediation operation is available
- exit code `2` when confirmation is required but prompting is unavailable, or when the user cancels

Non-interactive behavior:

- if a plan exists and `--diff` is not used, `--yes` is required in CI or `--no-interactive` mode

## `celador install [packages...]`

Run package installation with a preflight risk check.

```bash
celador install express
celador install left-pad --yes
```

Flags:

- `--yes` — continue on risky findings without prompting

Behavior:

- requires at least one package argument
- assesses the first package argument before execution
- fetches npm package metadata for npm-compatible managers
- inspects the release tarball for simple install-time risk heuristics
- delegates installation to the detected package manager after the assessment step

Current package-manager execution mapping:

- npm workspace → `npm install ...`
- pnpm workspace → `pnpm install ...`
- Bun workspace → `bun add ...`
- Deno workspace → install execution is unsupported in v1

Current risk heuristics include:

- `postinstall` scripts in package metadata
- `process.env` combined with network-related strings such as `http://`, `https://`, or `fetch(`
- unusually long hex-like strings

Non-interactive behavior:

- when the assessment requires approval, `--yes` is required outside a TTY

## `celador completion`

Cobra provides shell completion generation.

Supported shells:

- `bash`
- `zsh`
- `fish`
- `powershell`

Example:

```bash
celador completion bash
```
