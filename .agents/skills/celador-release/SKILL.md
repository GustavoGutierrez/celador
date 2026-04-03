---
name: celador-release
description: Publish a Celador GitHub release with GoReleaser assets and Homebrew tap synchronization. Trigger: When preparing or verifying a Celador release, tagging a version, or updating the tap branch.
---

# Celador Release Publishing

## When to use this skill

- Publishing a new Celador version
- Re-running a failed release for an existing tag
- Verifying GitHub release assets or the Homebrew tap branch

## Release source of truth

- Workflow: `.github/workflows/release.yml`
- GoReleaser config: `.goreleaser.yaml`
- Homebrew formula template: `homebrew-tap/Formula/celador.rb`

## Standard publish flow

1. Validate the repository locally:

   ```bash
   go test ./...
   go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml
   ```

2. Create and push the release tag:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

3. Let GitHub Actions run `.github/workflows/release.yml`.

## Manual rerun flow

Use this only when the tag already exists and the workflow needs to be rebuilt or retried:

```bash
gh workflow run release.yml -f tag=vX.Y.Z
```

The manual workflow expects an existing `v*` tag and will fail if the tag does not exist.

## Expected GitHub release assets

For tag `vX.Y.Z`, the release should include:

- `celador_X.Y.Z_linux_amd64.tar.gz`
- `celador_X.Y.Z_darwin_arm64.tar.gz`
- `celador_X.Y.Z_windows_amd64.zip`
- `checksums.txt`

## Homebrew behavior

- The workflow downloads `checksums.txt` from the GitHub release.
- It renders `homebrew-tap/Formula/celador.rb` with the current version and sha256 values.
- It creates or updates the `homebrew-tap` branch automatically.
- The published tap branch contains `Formula/celador.rb` at the repository root.

## Verification

Check release assets:

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/celador --json assets
```

Check the tap branch formula:

```bash
git fetch origin homebrew-tap
git show origin/homebrew-tap:Formula/celador.rb
```

Optional Homebrew install verification:

```bash
brew install https://raw.githubusercontent.com/GustavoGutierrez/celador/homebrew-tap/Formula/celador.rb
brew upgrade celador
celador --help
```

Homebrew does not support tapping a non-default branch directly, so Celador should be installed
from the raw formula URL published on the `homebrew-tap` branch.

## Failure handling

- If `goreleaser check` fails, fix the release configuration before tagging.
- If the release job succeeds but the tap update fails, rerun `release.yml` for the same tag.
- If `checksums.txt` is missing expected asset names, inspect `.goreleaser.yaml` archive naming first.
