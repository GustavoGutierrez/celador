---
name: celador-release
description: Publish a Celador GitHub release with GoReleaser assets and dedicated Homebrew tap synchronization. Trigger: When preparing or verifying a Celador release, tagging a version, or updating the tap repository.
---

# Celador Release Publishing

## When to use this skill

- Publishing a new Celador version
- Re-running a failed release for an existing tag
- Verifying GitHub release assets or the Homebrew tap repository

## Release source of truth

- Workflow: `.github/workflows/release.yml`
- GoReleaser config: `.goreleaser.yaml`
- Homebrew formula template: `packaging/homebrew/Formula/celador.rb`

## Standard publish flow

1. Validate the repository locally:

   ```bash
   go test ./...
   go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml
   ruby -c packaging/homebrew/Formula/celador.rb
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
- It renders `packaging/homebrew/Formula/celador.rb` with the current version and sha256 values.
- It pushes the rendered formula to `GustavoGutierrez/homebrew-celador`.
- The tap repository is consumed with `brew tap GustavoGutierrez/celador`.

## Verification

Check release assets:

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/celador --json assets
```

Check the tap repository formula:

```bash
gh repo view GustavoGutierrez/homebrew-celador
gh api repos/GustavoGutierrez/homebrew-celador/contents/Formula/celador.rb?ref=HEAD
```

Optional Homebrew install verification:

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
celador --help
```

Homebrew resolves `brew tap GustavoGutierrez/celador` to the dedicated repository
`GustavoGutierrez/homebrew-celador`, so this is the correct long-term install flow.

## Authentication requirement

The source repository workflow must use `HOMEBREW_TAP_GITHUB_TOKEN` to push to the separate tap
repository. A fine-grained token with `Contents: Read and write` on
`GustavoGutierrez/homebrew-celador` is sufficient.

## Failure handling

- If `goreleaser check` fails, fix the release configuration before tagging.
- If the release job succeeds but the tap update fails, rerun `release.yml` for the same tag after
  confirming `GustavoGutierrez/homebrew-celador` exists and `HOMEBREW_TAP_GITHUB_TOKEN` is valid.
- If `checksums.txt` is missing expected asset names, inspect `.goreleaser.yaml` archive naming first.
