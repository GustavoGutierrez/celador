# Release Process

This document explains how Celador releases and the Homebrew tap branch are published.

## Overview

Celador release automation is driven by:

- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `homebrew-tap/Formula/celador.rb`

The workflow publishes GitHub release assets and then creates or updates the `homebrew-tap`
branch in the same repository.

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

Local release config validation:

```bash
go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml
```

## Assets produced by the release workflow

For tag `vX.Y.Z`, GoReleaser publishes:

- `celador_X.Y.Z_linux_amd64.tar.gz`
- `celador_X.Y.Z_darwin_arm64.tar.gz`
- `celador_X.Y.Z_windows_amd64.zip`
- `checksums.txt`

These assets are attached directly to the GitHub release for the same tag.

## Homebrew formula update behavior

After the release assets are published, the workflow:

1. Downloads `checksums.txt` from the GitHub release
2. Replaces the placeholders in `homebrew-tap/Formula/celador.rb`
3. Publishes the rendered formula to the `homebrew-tap` branch as `Formula/celador.rb`

If the `homebrew-tap` branch does not exist yet, the workflow creates it automatically.

## Verification commands

After the workflow completes, verify the release with:

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/celador --json assets
```

Confirm that the release includes the three platform archives and `checksums.txt`.

Then verify the Homebrew branch:

```bash
git fetch origin homebrew-tap
git show origin/homebrew-tap:Formula/celador.rb
```

Optionally test installation with Homebrew:

```bash
brew install https://raw.githubusercontent.com/GustavoGutierrez/celador/homebrew-tap/Formula/celador.rb
brew upgrade celador
celador --help
```

This repository uses a same-repo `homebrew-tap` branch, and Homebrew does not support tapping a
non-default branch directly with `brew tap`, so the raw formula URL is the supported install path.

## Troubleshooting

### The workflow fails on manual dispatch

Make sure the tag already exists on the remote:

```bash
git ls-remote --tags origin "vX.Y.Z"
```

### The formula does not update

Check whether `checksums.txt` contains these asset names exactly:

- `celador_X.Y.Z_darwin_arm64.tar.gz`
- `celador_X.Y.Z_linux_amd64.tar.gz`

### The tap branch is missing

The release workflow creates `homebrew-tap` on first publish. Re-run the release workflow for
the tag if the first branch publish failed.

## Release checklist

- [ ] Run `go test ./...`
- [ ] Run `go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml`
- [ ] Push the release tag `vX.Y.Z`
- [ ] Wait for `release.yml` to finish
- [ ] Confirm GitHub release assets exist
- [ ] Confirm `homebrew-tap` contains `Formula/celador.rb`
- [ ] Verify `brew install celador` from the tap branch
