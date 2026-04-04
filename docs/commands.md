# Command Reference

## Root command

```text
celador
```

Global flag:

- `--no-interactive` — disables prompts and TTY flows
- `--version` — prints the current CLI version and, when reachable, whether a newer release is available

Available commands:

- `init`
- `scan`
- `fix`
- `install`
- `about`
- `tui`
- `completion`

Version behavior:

- `celador --version` always prints the current CLI version
- when GitHub release metadata is reachable, it also reports whether a newer release exists
- when Celador appears to be installed from Homebrew, it recommends:

  ```bash
  brew update && brew upgrade GustavoGutierrez/celador/celador
  ```

- if the release lookup fails, the command still succeeds and prints the current version only

## `celador about`

Show project, release, and developer information in plain text.

```bash
celador about
```

Behavior:

- shows the developer name and GitHub profile
- shows the current installed version
- shows the latest available release when GitHub release metadata is reachable
- summarizes the primary commands and documentation entry points

## `celador tui`

Open the interactive Celador overview.

```bash
celador tui
celador --no-interactive tui
```

Behavior:

- uses Bubble Tea and Lip Gloss when an interactive TTY is available
- shows the current version and latest available release when known
- shows upgrade guidance for Homebrew users when an update is available
- includes a concise command reference and documentation pointers
- falls back to a plain-text overview in CI or when `--no-interactive` is used

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
- appends missing recommended entries to `.gitignore` and `.npmignore`
- applies package-manager hardening config
- creates or updates managed guidance in `AGENTS.md`
- updates managed guidance in `CLAUDE.md` only when that file already exists
- requires a strict `package.json` `engines.node` entry for `package.json` workspaces
- can add a missing `engines.node` automatically from the current local Node.js version

Important constraints:

- does not accept positional arguments
- fails if no supported workspace lockfile is found
- for `package.json` workspaces, fails when `engines.node` is missing and Celador cannot detect a local Node.js version to add automatically
- prompts before adding a missing `engines.node` in interactive mode when local Node.js detection succeeds
- auto-adds a missing `engines.node` in CI or `--no-interactive` mode when local Node.js detection succeeds
- fails if `package.json` `engines.node` is not a strict exact version such as `20.11.1`

## `celador scan`

Audit supported lockfiles and active framework rules.

```bash
celador scan
celador scan --json
celador scan --verbose
celador --no-interactive scan
```

Flags:

- `--json` — render structured JSON output for tooling while preserving the usual scan exit codes
- `--verbose` — render extra scan metadata in text mode

Behavior:

- requires at least one supported lockfile
- parses dependencies from the detected lockfile set
- loads ignore rules from `.celadorignore`
- loads built-in YAML rule packs from `configs/rules`
- queries OSV.dev with cache reuse and offline fallback behavior
- renders grouped plain-text findings by severity in default mode
- can render structured JSON for CI integrations and custom tooling

Exit behavior:

- exit code `0` when no findings remain after ignores
- exit code `3` when findings are detected
- exit code `2` for invalid CLI usage

Output behavior:

- prints a scan fingerprint
- prints the finding count and ignored finding count
- groups default text output by severity for easier review
- prints one line per finding with the finding ID, package or target context, richer advisory text when available, and `fixed in` details when known
- in `--verbose` mode, also prints dependency count, package manager, and active rule-pack version
- in `--json` mode, includes fingerprint, cache flags, workspace metadata, rendered and raw finding counts, and detailed finding records
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
- bumps existing dependencies when possible, including `dependencies`, `devDependencies`, `optionalDependencies`, and `peerDependencies`
- otherwise writes package overrides
- explains why no safe plan was generated when findings cannot be remediated conservatively
- separates at least these unplanned categories when present:
  - findings without a known fixed version
  - rule or code findings that require manual changes
  - findings outside the current remediation scope
- prints a readable operation list before the dry-run diff when a plan exists

Current scope:

- does not update lockfiles directly
- does not run a package-manager install after writing `package.json`
- returns a dry-run diff preview for the plan or explains why no manifest diff can be produced

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
- resolves the package manager from the workspace lockfile when present, otherwise from practical workspace signals such as `package.json`, `packageManager`, `pnpm-workspace.yaml`, `bunfig.toml`, or Deno config files
- defaults to npm for `package.json` workspaces when no stronger manager signal is present
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
