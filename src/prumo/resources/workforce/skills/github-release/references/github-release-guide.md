# GitHub Release Packaging — Technical Reference Guide

## 1. Core Concepts

**SemVer discipline**: breaking API/config change → major; additive functionality → minor; fixes only → patch. Prereleases (`-rc.N`) never mutate; published tags are immutable — mistakes ship as new patches, never force-moved tags.

**Generated deltas**: changelog derives from merged PRs (Features / Fixes / Breaking with PR+issue links), e.g. `gh api repos/{owner}/{repo}/releases/generate-notes -f tag_name=v0.5.0`. Hand-written entries are for upgrade impact only.

**Asset integrity**: CI builds from the exact tag commit; every downloadable artifact is covered by a published `SHA256SUMS` file plus an SBOM (CycloneDX/SPDX). Hand-uploaded binaries are prohibited.

## 2. Patterns

- **Tag-then-build**: `git tag -a v0.5.0 -m "..." && git push origin v0.5.0`; release workflow triggers on the tag, builds the matrix, attaches assets, publishes checksums.
- **Upgrade-impact lead**: notes open with breaking changes + migration steps, then highlights, then full changelog link.
- **Tag-SHA proof**: `gh release view v0.5.0 --json tagCommit` must equal the intended commit SHA; verify before announcing.
- **Checksum verification**: `sha256sum -c SHA256SUMS` on downloaded assets from a clean machine; log the output as evidence.

## 3. Anti-Patterns

- Force-moving a published tag to "fix" a release.
- Release notes with no migration steps for breaking changes.
- Attaching locally-built binaries ("works on my machine" artifacts).
- Cutting releases from unmerged branches or dirty working trees.

## 4. Worked Example

Release v0.5.0: 14 PRs since v0.4.2 → generated notes grouped 6 features / 7 fixes / 1 breaking (`config.timeout_ms` renamed to `config.timeout`; migration: rename key, default 5000 unchanged). CI built linux/darwin/windows binaries + `SHA256SUMS` + `sbom.cdx.json`. Tag-SHA verified (`9f2c...` matches `main` merge commit). Notes lead with the breaking change and migration snippet. Checksum log archived in the task report.

## 5. Verification Pointers

- `gh release view` shows expected assets + checksums file.
- Tag target SHA equals the intended commit; no force-push in tag history.
- Breaking changes (if any) carry migration steps; otherwise an explicit "no breaking changes" line.
