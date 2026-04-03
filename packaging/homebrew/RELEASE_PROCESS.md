# Release Process

This document explains how Celador releases and the dedicated Homebrew tap are published.

## Overview

Celador release automation is driven by:

- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `packaging/homebrew/Formula/celador.rb`
- `scripts/render_homebrew_formula.py`

The workflow publishes GitHub release assets in `GustavoGutierrez/celador` and then updates the
dedicated tap repository `GustavoGutierrez/homebrew-celador`.

## Release trigger

Preferred trigger:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

Optional manual rerun for an existing tag:

```bash
gh workflow run release.yml -f tag=vX.Y.Z
```

The manual workflow does not create tags. It only rebuilds or republishes an existing tag.

## Local validation

```bash
go test ./...
go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml
ruby -c packaging/homebrew/Formula/celador.rb
```

## Assets produced by the release workflow

For tag `vX.Y.Z`, GoReleaser publishes:

- `celador_X.Y.Z_linux_amd64.tar.gz`
- `celador_X.Y.Z_darwin_arm64.tar.gz`
- `celador_X.Y.Z_windows_amd64.zip`
- `checksums.txt`

These assets are attached directly to the GitHub release for the same tag.

## Homebrew publish behavior

After the release assets are published, the workflow:

1. Downloads `checksums.txt` from the GitHub release
2. Renders `packaging/homebrew/Formula/celador.rb`
3. Pushes `Formula/celador.rb` and tap docs to `GustavoGutierrez/homebrew-celador`

Because the tap repository follows Homebrew naming conventions, users can install Celador with:

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
```

`brew tap GustavoGutierrez/celador` is correct here specifically because Homebrew resolves it to the
repository `GustavoGutierrez/homebrew-celador`.

## Authentication requirement

The workflow needs a cross-repository credential to update the tap repository.

Configure a secret named `HOMEBREW_TAP_SSH_KEY` in `GustavoGutierrez/celador`.

Recommended setup:

- Create a dedicated write-enabled deploy key on `GustavoGutierrez/homebrew-celador`
- Store the private key as the `HOMEBREW_TAP_SSH_KEY` repository secret in `GustavoGutierrez/celador`

The default `GITHUB_TOKEN` from the source repository is not sufficient for this cross-repository
push.

## Verification commands

After the workflow completes, verify the release with:

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/celador --json assets
```

Confirm that the release includes the three platform archives and `checksums.txt`.

Then verify the tap repository:

```bash
gh repo view GustavoGutierrez/homebrew-celador
gh api repos/GustavoGutierrez/homebrew-celador/contents/Formula/celador.rb?ref=HEAD
```

Optionally test installation with Homebrew:

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
celador --help
brew info GustavoGutierrez/celador/celador
```

## Troubleshooting

### The workflow fails before the tap update

Make sure the tag already exists on the remote:

```bash
git ls-remote --tags origin "vX.Y.Z"
```

### The formula does not update

Check whether `checksums.txt` contains these asset names exactly:

- `celador_X.Y.Z_darwin_arm64.tar.gz`
- `celador_X.Y.Z_linux_amd64.tar.gz`

### The tap repository does not exist yet

Create `GustavoGutierrez/homebrew-celador` before running the release workflow. The workflow updates
that repository, but it does not create GitHub repositories automatically.

### The tap publish step cannot authenticate

Confirm that `HOMEBREW_TAP_SSH_KEY` exists and matches a write-enabled deploy key on the tap repository.

## Release checklist

- [ ] Run `go test ./...`
- [ ] Run `go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml`
- [ ] Run `ruby -c packaging/homebrew/Formula/celador.rb`
- [ ] Ensure `GustavoGutierrez/homebrew-celador` exists
- [ ] Ensure `HOMEBREW_TAP_SSH_KEY` is configured
- [ ] Push the release tag `vX.Y.Z`
- [ ] Wait for `release.yml` to finish
- [ ] Confirm GitHub release assets exist
- [ ] Confirm the tap repository contains `Formula/celador.rb`
- [ ] Verify Homebrew install from the tap
