# 🛡️ Celador CLI

> "The security deadlock for your dependencies."

Celador is an ultra-fast, zero-trust supply chain security CLI for modern JavaScript/TypeScript ecosystems. Written in Go and powered by a beautiful Bubble Tea terminal interface, Celador proactively scans, blocks, and remediates malicious packages, framework misconfigurations, and code vulnerabilities (like SQLi and XSS) before they infect your system.

## 🚀 Features

- **`celador init` (Guided Setup):** Instantly integrates Celador into your project, automatically detects your package manager, injects hardening rules into `.npmrc`/`bunfig`, and sets up Git hooks to prevent committing vulnerable code.
- **`celador install` (Zero-Trust Wrapper):** Automatically detects your package manager, downloads tarballs into a temporary sandbox, and performs heuristic analysis (detecting network requests with env vars) *before* installing anything.
- **`celador scan` (Lightning Fast OSV Scanning):** Uses Google's OSV.dev batch API combined with a local lockfile-hash cache fallback to audit your dependencies in milliseconds without unnecessary network overhead.
- **Framework Fingerprinting & SAST:** 
  - Validates **Next.js**, **Nuxt.js**, **SvelteKit**, and **Strapi** config files to prevent source code leaks, SSRF, and default cryptographic keys.
  - Mitigates **Tailwind CSS v4** XSS risks by detecting dynamic arbitrary value interpolation (`bg-[${input}]`).
  - Detects **SQL Injection** vulnerabilities in raw database queries.
- **Proactive Hardening:** 
  - Disables arbitrary `postinstall` scripts (`ignore-scripts=true`).
  - Sets `minimumReleaseAge: 1440` (24 hours) to prevent Day-0 hijacked package installations.
  - Automatically fixes `.gitignore` to prevent leaking `.env.local` and sourcemaps (`*.map.js`).
- **AI Agent Guidelines:** Automatically provisions `AGENTS.md` and `CLAUDE.md` with strict rules to prevent AI coding assistants from introducing vulnerable dependency patterns, and includes an `llm.txt` file for complete AI context.
- **Smart Remediation:** Use `celador fix --auto` to apply Safe SemVer bumps, or use `celador fix --pr` to automatically generate Git branches and Pull Requests.

## 📦 Installation

Since Celador is a pre-compiled Go binary, installation is instant:

```bash
npm install -g celador-cli
# or via Homebrew
brew install celador
# or via universal script
curl -fsSL https://celador.dev/install.sh | sh
```

## 🛠️ Usage

### 1. Initialize
Set up Celador in an existing project and configure your pre-commit hooks:
```bash
celador init
```

### 2. The Safe Install
Replace your daily `npm install` with Celador's secure wrapper:
```bash
celador install express
```

### 3. Audit and Fix
Run an interactive audit utilizing our beautiful TUI, and fix issues proactively:
```bash
celador scan --staged
celador fix --interactive
celador fix --pr # Automatically create Git branches and PRs for fixes
```

### 4. Generate Reports
Export to standard enterprise formats for CI/CD pipelines (SARIF, CycloneDX with VEX):
```bash
celador report --format=sarif --out=security.sarif
```

## 🏗️ Architecture & Contributing

Celador is built using **Hexagonal Architecture (Ports & Adapters)** in Go. 
- The project strictly follows **Spec Driven Development (SDD)**. Specs and behavioral tests must be written before implementation.
- All codebase elements, comments, and internal Software Design Documents are strictly written in **English**.
- The UI is powered by `charmbracelet/bubbletea` and CLI routing by `spf13/cobra`.
- Follows standard Go project layout and strict naming conventions.
- Comprehensive Unit Tests cover all domains.

## 📄 License
MIT License.
