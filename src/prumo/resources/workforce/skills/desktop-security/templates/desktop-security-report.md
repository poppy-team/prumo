# Desktop Security Audit — Document Opener

## Scope
- **Component**: main-process document opener and custom protocol handler
- **Platforms**: Windows 11 and macOS 15
- **Review date**: 2026-09-23
- **Assessor**: desktop-security-agent

## Finding DESK-01
- **Severity**: Critical
- **Observed**: The `open-file` IPC command accepted an absolute path without checking it against the approved document root.
- **Impact**: A compromised renderer could ask the privileged process to open arbitrary local files.
- **Remediation**: Resolve links, compare canonical roots, validate type and size, and use the sandboxed parser.
- **Regression test**: Submit `/etc/passwd`, `../../secrets.pdf`, and an escaping symlink; all must be denied.

## Finding DESK-02
- **Severity**: High
- **Observed**: Update installation checked HTTPS but not the detached binary signature before extraction.
- **Remediation**: Pin the vendor key, verify manifest and binary signatures, then verify digest and version monotonicity.

## Verified Controls
- IPC uses an allowlist and rejects unknown fields.
- Renderer has no raw shell or filesystem capability.
- OAuth tokens use the OS credential vault and are absent from renderer storage.
- The updater aborts on any signature mismatch.

## Release Decision
DESK-01 and DESK-02 block release. Require the traversal, malformed-message, and tampered-update suites to pass.
