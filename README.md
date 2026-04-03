<p align="center">
  <img src="celador.png" width="500" alt="Celador CLI" />
</p>

# 🛡️ Celador CLI

> "The security deadlock for your dependencies."

Celador is a zero-trust supply chain security CLI for modern JavaScript, TypeScript, and Deno workspaces. Written in Go, Celador scans dependencies, flags risky framework configuration, and helps apply conservative remediations with deterministic non-interactive behavior.

## 🚀 Features

- **`celador init`:** Detects the current workspace, writes Celador config, hardens package-manager settings, updates ignore hygiene files, and refreshes managed guidance files.
- **`celador scan`:** Audits supported lockfiles with OSV-backed dependency findings, built-in framework rules, ignore handling, and persistent cache reuse.
- **`celador fix`:** Plans and applies conservative manifest-level remediation to `package.json`.
- **`celador install`:** Assesses install-time package risk before delegating to npm, pnpm, or Bun.
- **Plain-text, deterministic behavior:** Non-interactive flows are supported through `--no-interactive` and CI-aware prompting rules.

## 📚 Documentation

See the docs index for the full current CLI documentation:

- [`docs/README.md`](docs/README.md)
- [`docs/installation.md`](docs/installation.md)
- [`docs/commands.md`](docs/commands.md)
- [`docs/configuration.md`](docs/configuration.md)
- [`docs/security-rules.md`](docs/security-rules.md)
- [`docs/release-packaging.md`](docs/release-packaging.md)
- [`docs/TECHNICAL_SUMMARY.md`](docs/TECHNICAL_SUMMARY.md)

## 📦 Installation

### macOS and Linux (Homebrew)

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
```

### Windows

Download the Windows release asset from GitHub Releases:

- <https://github.com/GustavoGutierrez/celador/releases>
- `celador_X.Y.Z_windows_amd64.zip`

## 🛠️ Quick Usage

```bash
celador init
celador scan
celador fix --diff
celador fix --yes
celador install express
```

## 🏗️ Architecture & Contributing

Celador is a Go CLI built with **hexagonal architecture (ports and adapters)**.

- CLI routing is powered by `spf13/cobra`.
- The default runtime surface is plain-text output.
- Project standards and contributor guidance live in [`AGENTS.md`](AGENTS.md).
- Architecture details live in [`docs/architecture.md`](docs/architecture.md).

## 📄 License
MIT License.
