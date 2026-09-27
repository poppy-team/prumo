# Untrusted Project Security Clearance Report

## Scope
- Repository: `acme-widgets` source archive
- Assessor: quarantine-review-agent
- Intake date: 2026-09-23
- Analysis mode: static, no execution, network egress disabled

## Findings
- **CRITICAL**: `package.json` defines a `postinstall` hook that downloads an executable from an external host.
- **HIGH**: `assets/logo.svg` contains a 4 KB base64 blob with a non-image magic prefix.
- **MEDIUM**: `.vscode/tasks.json` launches a shell command on workspace open.

## Static Evidence
- Trigger paths listed: `.git/hooks/*`, `.vscode/tasks.json`, workflows, `Makefile`, and npm lifecycle scripts.
- Dependency manifests were parsed as text; no package manager or build ran.
- Egress counters remained at zero during the inspection window.

## Verdict
`QUARANTINED`, risk score 9/10. Human maintainer approval and an isolated read-only sandbox are required before any dynamic build. The original archive remains outside the active repository.
