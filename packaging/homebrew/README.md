# Celador Homebrew Tap Assets

This directory contains the source files that Celador publishes to the dedicated Homebrew tap
repository:

- Tap repository: `GustavoGutierrez/homebrew-celador`
- Tap command: `brew tap GustavoGutierrez/celador`

Homebrew maps `brew tap <owner>/<name>` to the GitHub repository
`<owner>/homebrew-<name>`, so `brew tap GustavoGutierrez/celador` resolves to
`GustavoGutierrez/homebrew-celador`.

## Supported Homebrew targets

- macOS arm64 (Apple Silicon)
- Linux amd64

The GitHub release workflow also publishes a Windows amd64 archive, but Homebrew does not use it.

## Install Celador with Homebrew

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
```

After installation, upgrades use the normal Homebrew flow:

```bash
brew update
brew upgrade celador
```

## How publishing works

On every tagged release (`v*`) the GitHub Actions release workflow:

1. Publishes release archives and `checksums.txt` to `GustavoGutierrez/celador`
2. Renders `Formula/celador.rb` from `packaging/homebrew/Formula/celador.rb`
3. Commits the rendered formula and tap docs to `GustavoGutierrez/homebrew-celador`

## Authentication requirement

Cross-repository updates cannot rely on the source repository's default `GITHUB_TOKEN`.

Create a repository secret named `HOMEBREW_TAP_SSH_KEY` in `GustavoGutierrez/celador`.

That secret must contain the private half of a write-enabled deploy key registered on
`GustavoGutierrez/homebrew-celador`.
