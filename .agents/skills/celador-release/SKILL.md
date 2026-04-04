---
name: celador-release
description: Publish a Celador GitHub release with GoReleaser assets and dedicated Homebrew tap synchronization. Trigger: When preparing or verifying a Celador release, tagging a version, or updating the tap repository.
---

# Celador Release Publishing

## When to use this skill

- Publishing a new Celador version
- Re-running a failed release for an existing tag
- Verifying GitHub release assets or the dedicated Homebrew tap after the tap migration

## Release source of truth

- Workflow: `.github/workflows/release.yml`
- GoReleaser config: `.goreleaser.yaml`
- Homebrew formula template: `packaging/homebrew/Formula/celador.rb`
- Homebrew tap docs: `packaging/homebrew/README.md`, `packaging/homebrew/RELEASE_PROCESS.md`

## Dedicated tap decision

- User-facing Homebrew commands stay:

  ```bash
  brew tap GustavoGutierrez/celador
  brew install GustavoGutierrez/celador/celador
  ```

- Homebrew resolves that tap command to the dedicated repository
  `GustavoGutierrez/homebrew-celador`.
- Windows is not distributed through Homebrew; Windows users install from the GitHub release asset.

## Versioning rule

- Always use Semantic Versioning for Celador releases: `MAJOR.MINOR.PATCH`.
- Tag format must always be `vMAJOR.MINOR.PATCH`, for example `v0.1.1`.
- Use:
  - `PATCH` for documentation, release automation, packaging, and backward-compatible fixes.
  - `MINOR` for backward-compatible features.
  - `MAJOR` for breaking CLI, config, or workflow changes.

### Automatic version deduction when the user says "publish a new release"

When the user asks to publish a new release without specifying a version, inspect the unreleased changes and
choose the next version automatically:

- Bump `PATCH` for backward-compatible fixes, docs, packaging, release automation, and maintenance updates.
- Bump `MINOR` for backward-compatible user-facing features, new command capabilities, richer CLI flows, or
  notable UX additions.
- Bump `MAJOR` only for breaking CLI, config, workflow, or compatibility changes.

Always explain the chosen bump briefly before tagging.

## Full operator flow for "publish a new release"

When the user says some variation of "publish a new release", execute this full flow end to end unless the
user explicitly narrows the scope:

1. Review the working tree and unreleased changes.
2. Deduce the next Semantic Version if the user did not provide one.
3. Run local release validation.
4. Create a release commit if there are uncommitted changes and the user asked to publish.
5. Create the release tag `vX.Y.Z`.
6. Push the commit and tag.
7. Let `.github/workflows/release.yml` publish GitHub assets and update the dedicated Homebrew tap.
8. Verify the GitHub release assets.
9. Verify the Homebrew tap publication.
10. Report the final release URL, tag, and verification outcome.

## Standard publish flow

1. Validate the repository locally before tagging:

   ```bash
   go test ./...
   go run github.com/goreleaser/goreleaser/v2@v2.8.2 check --config .goreleaser.yaml
   ruby -c packaging/homebrew/Formula/celador.rb
   ```

2. Ensure the dedicated tap repository exists and the cross-repository credential is configured:

   - Tap repository: `GustavoGutierrez/homebrew-celador`
   - Required secret in `GustavoGutierrez/celador`: `HOMEBREW_TAP_SSH_KEY`
   - That secret must be the private half of a write-enabled deploy key registered on
     `GustavoGutierrez/homebrew-celador`

3. If the release includes local changes that are not yet committed, create a normal git commit first.

4. Create and push the release tag:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

5. Let GitHub Actions run `.github/workflows/release.yml`.

## What the workflow does

For a tagged release, `.github/workflows/release.yml`:

1. Publishes GitHub release assets from GoReleaser
2. Downloads `checksums.txt`
3. Renders `packaging/homebrew/Formula/celador.rb` with the version and sha256 values
4. Validates the rendered formula with Ruby
5. Checks out `GustavoGutierrez/homebrew-celador` over SSH
6. Publishes `Formula/celador.rb` plus tap docs into the dedicated tap repository

## Manual rerun flow

Use this only when the tag already exists and the workflow needs to be rebuilt or retried:

```bash
gh workflow run release.yml -f tag=vX.Y.Z
```

Important rerun rule:

- `workflow_dispatch` does **not** create the tag
- The `vX.Y.Z` tag must already exist remotely
- Reruns are the correct recovery path when release assets or the tap publish step must be rebuilt for
  the same version

## Expected GitHub release assets

For tag `vX.Y.Z`, the release should include:

- `celador_X.Y.Z_linux_amd64.tar.gz`
- `celador_X.Y.Z_darwin_arm64.tar.gz`
- `celador_X.Y.Z_windows_amd64.zip`
- `checksums.txt`

Windows users should consume `celador_X.Y.Z_windows_amd64.zip` directly from GitHub Releases.

## Homebrew tap publishing details

- The release workflow updates the dedicated tap repository, not the source repository.
- Published tap contents should include:
  - `Formula/celador.rb`
  - `README.md`
  - `RELEASE_PROCESS.md`
- The tap is installed by users with:

  ```bash
  brew tap GustavoGutierrez/celador
  brew install GustavoGutierrez/celador/celador
  ```

## Verification

### Verify GitHub release assets

```bash
gh release view vX.Y.Z --repo GustavoGutierrez/celador --json assets
```

Confirm the release includes the expected Linux, macOS, and Windows archives plus `checksums.txt`.

### Verify the dedicated tap repository

```bash
gh repo view GustavoGutierrez/homebrew-celador
gh api repos/GustavoGutierrez/homebrew-celador/contents/Formula/celador.rb?ref=HEAD
```

### Verify Homebrew installation behavior

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
celador --help
brew info GustavoGutierrez/celador/celador
```

## Authentication requirement

The source repository workflow must use `HOMEBREW_TAP_SSH_KEY` to push to the separate tap
repository. The default source-repository `GITHUB_TOKEN` is not sufficient for this cross-repository
push.

## Failure handling

- If `goreleaser check` fails, fix the release configuration before tagging.
- If the tag does not exist remotely, create or push the tag before using the manual rerun flow.
- If GitHub release assets are missing, inspect `.goreleaser.yaml` and rerun `release.yml` for the
  existing tag.
- If the release job succeeds but the tap update fails, rerun `release.yml` for the same tag after
  confirming `GustavoGutierrez/homebrew-celador` exists and `HOMEBREW_TAP_SSH_KEY` is valid.
- If `checksums.txt` is missing expected asset names, inspect `.goreleaser.yaml` archive naming first.
