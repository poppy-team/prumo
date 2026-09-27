# Desktop Security Verification Checklist

## Trust Boundaries
- [ ] Renderer, main, helpers, protocol handlers, updater, keychain, and network boundaries are mapped.
- [ ] Each privileged capability has an owner, input schema, caller, and abuse cases.
- [ ] The renderer is treated as untrusted and cannot invoke raw shell or filesystem operations.

## IPC and External Input
- [ ] IPC channels are allowlisted and privileged-side schemas reject unknown fields.
- [ ] Deep links, opened files, drag-and-drop data, and protocol URLs are parsed strictly.
- [ ] Paths are resolved beneath approved roots; traversal and escaping links are denied.

## Updates and Sandbox
- [ ] Manifest and binary signatures verify against pinned keys before extraction or execution.
- [ ] Update version and channel are monotonic and failed verification aborts installation.
- [ ] Entitlements, sandbox profiles, subprocesses, and filesystem access are least-privilege.

## Secrets and Audit
- [ ] Secrets use the OS credential vault and never enter renderer storage, argv, or logs.
- [ ] Security events include actor, boundary, result, and correlation ID without secret payloads.
- [ ] Failure logs are sanitized but sufficient to diagnose rejected operations.

## Abuse Evidence
- [ ] Malformed IPC, traversal, deep-link, tampered-update, and sandbox cases are automated.
- [ ] No release-blocking escape or update-spoofing vector remains.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
