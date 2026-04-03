# Release and Packaging Overview

This page gives contributors a high-level release view. For maintainer depth, keep using:

- `packaging/homebrew/README.md`
- `packaging/homebrew/RELEASE_PROCESS.md`
- `.goreleaser.yaml`
- `.github/workflows/release.yml`

## Release model

Celador publishes GitHub release assets from tagged versions in the form `vMAJOR.MINOR.PATCH`.

The release pipeline is driven by:

- `.github/workflows/release.yml`
- `.goreleaser.yaml`

## Current packaged artifacts

For each release tag `vX.Y.Z`, the pipeline publishes:

- `celador_X.Y.Z_linux_amd64.tar.gz`
- `celador_X.Y.Z_darwin_arm64.tar.gz`
- `celador_X.Y.Z_windows_amd64.zip`
- `checksums.txt`

## Homebrew distribution

Homebrew publishing uses the dedicated tap repository:

- repository: `GustavoGutierrez/homebrew-celador`
- user tap command: `brew tap GustavoGutierrez/celador`

User installation flow:

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
```

## Windows distribution

Windows is distributed through the GitHub release asset, not through Homebrew.

## What the release workflow does

At a docs level, the workflow:

1. validates the tag and release metadata
2. runs GoReleaser to build and publish release assets
3. downloads `checksums.txt`
4. renders the Homebrew formula
5. commits the formula and tap docs to `GustavoGutierrez/homebrew-celador`

## Publishing requirement

Tap publishing requires the repository secret:

- `HOMEBREW_TAP_SSH_KEY`

That secret must contain the private half of a write-enabled deploy key registered on `GustavoGutierrez/homebrew-celador`.
