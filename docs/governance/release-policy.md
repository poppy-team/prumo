# Release Policy

## Versioning

Use Semantic Versioning-compatible release tags:

```text
v0.6.0-alpha.1
v0.6.0-beta.1
v0.6.0-rc.1
v0.6.0
v1.0.0
```

Published tags are immutable. Agents must not delete, move, recreate, or force-update a published release tag.

## Release branch and PR

Release work uses:

```text
release/<version>
```

Release changes enter `main` through a Pull Request with:

- release notes;
- migration impact;
- validation evidence;
- binary/checksum results;
- rollback plan.

## Artifacts

`sh scripts/release.sh` produces:

- Linux amd64/arm64;
- macOS amd64/arm64;
- Windows amd64/arm64;
- `checksums.txt`;
- `release.json`.

The release workflow uploads these artifacts on `v0.6.*` tags.

## Promotion

Release progression:

```text
0.5.0-alpha → 0.5.0-beta → 0.5.0-rc → 0.5.0
```

Promotion requires the relevant phase gate, full Go checks, Python oracle checks while migration is active, conformance, and release smoke tests.

## Privileged actions

Publishing a release is privileged. Agents may prepare artifacts and a Pull Request, but release publication requires explicit authorized workflow and evidence. No normal agent workflow may mutate an existing release tag.
