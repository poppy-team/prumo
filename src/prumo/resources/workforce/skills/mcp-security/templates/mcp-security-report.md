# MCP Security Audit — Workspace Tool Gateway

## Scope
- **Component**: MCP tool gateway for repository search, file read, and issue creation
- **Review date**: 2026-09-23
- **Assessor**: mcp-security-agent
- **Server trust**: untrusted until capability contract and binary digest are pinned

## Finding MCP-01
- **Severity**: Critical
- **Observed**: The `read_file` tool accepted any absolute path.
- **Impact**: A client could request `.env`, private keys, and OS credential files.
- **Remediation**: Resolve paths, deny protected roots, and authorize only declared workspace paths.
- **Regression test**: `.env`, `.git/config`, `/root/.ssh/id_ed25519`, traversal, and escaping symlink all return `E_ACCESS_DENIED`.

## Finding MCP-02
- **Severity**: High
- **Observed**: Retrieved issue text containing “ignore previous instructions” was returned as an untyped string.
- **Remediation**: Mark all tool results as untrusted data, enforce result schemas, and prevent content from granting capabilities.

## Control Verification
- Every mutating call records sanitized intent before execution and outcome afterward.
- The gateway enforces a 30-second timeout and 1 MiB output cap.
- Subprocess environment contains no master token or cloud credentials.
- Issue publication requires approval bound to repository, title, and body digest.

## Release Decision
MCP-01 and MCP-02 block release. Require path, injection, approval, and resource-bound tests to pass.
