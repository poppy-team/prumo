# GitHub Release Packaging — Verification Checklist

## 1. Version Intent & Delta
- [ ] SemVer bump classified from merged changes (breaking → major, feature → minor, fixes → patch).
- [ ] Changelog generated from merged PRs grouped Features/Fixes/Breaking with PR and issue links.
- [ ] Prerelease suffixes (`-rc.N`) used for candidates; no published tag mutated.

## 2. Build & Asset Integrity
- [ ] Tag created annotated from the intended commit; CI built assets from that exact commit.
- [ ] Asset matrix complete (per-platform binaries, `SHA256SUMS`, SBOM in CycloneDX/SPDX).
- [ ] Checksums verified on clean-machine downloads with log archived; no hand-uploaded binaries.

## 3. Notes & Upgrade Guidance
- [ ] Notes lead with upgrade impact; breaking changes carry migration steps or release is blocked.
- [ ] Supported upgrade paths stated; full changelog linked.
- [ ] Tag target SHA proof (`gh release view --json tagCommit`) matches the intended commit.

## 4. Evidence & Sign-Off
- [ ] Release URL, CI run, and checksum log linked in the task report.
- [ ] Post-publish rendering of notes and asset list confirmed.
- [ ] `scripts/verify.sh` exits 0 from the repository root.
