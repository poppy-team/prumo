# Release v0.5.0

## Version Intent
- Previous tag: `v0.4.2`
- Target: `v0.5.0`
- Classification: minor additive release with one breaking configuration rename.

## Upgrade Impact
`config.timeout_ms` is now `config.timeout`; the default remains 5000 ms. Update deployment configuration before upgrading.

## Assets
- `prumo-linux-x86_64`
- `prumo-darwin-arm64`
- `prumo-windows-x86_64`
- `SHA256SUMS`
- `sbom.cdx.json`

## Evidence
- Tag target: `9f2c1a7e4b6d`
- CI run: `release-1849`
- Checksum verification: all assets passed
- Full changelog: generated from 14 merged PRs
