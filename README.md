# Celador Homebrew Tap

This directory contains the files that are published to the `homebrew-tap` branch for
[Celador](https://github.com/GustavoGutierrez/celador).

The release workflow copies these files to the root of the `homebrew-tap` branch and
replaces the formula placeholders with the current release version and checksums.

## Supported Homebrew targets

- macOS arm64 (Apple Silicon)
- Linux amd64

The GitHub release workflow also publishes a Windows amd64 archive, but Homebrew does not
use it.

## Install directly from the formula URL

Homebrew cannot tap a non-default branch directly. Because this formula is published to the
`homebrew-tap` branch of the same repository, the supported install path is the raw formula URL:

```bash
brew install https://raw.githubusercontent.com/GustavoGutierrez/celador/homebrew-tap/Formula/celador.rb
```

After installation, standard upgrades can use:

```bash
brew upgrade celador
```

## Update

```bash
brew update
brew upgrade celador
```

## Verify the installed binary

```bash
celador --help
brew info celador
```

## How the branch is maintained

On every tagged release (`v*`) the GitHub Actions release workflow:

1. Publishes release archives and `checksums.txt`
2. Renders `Formula/celador.rb` with the new version and sha256 values
3. Creates or updates the `homebrew-tap` branch in this repository

If the workflow is rerun for the same tag, the branch update is idempotent and simply
replaces the formula when needed.
