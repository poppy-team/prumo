# Desktop Security Reference Guide

## Trust Boundaries

Map renderer, main process, helper processes, protocol handlers, opened files, update feeds, updater, keychain, and network as separate boundaries. Renderer content is untrusted even when loaded locally. Security checks execute only in the privileged side.

## IPC Validation

Allowlist channels and schemas. Validate types, ranges, lengths, enum values, origin, and requested privilege before dispatch. A command such as `openDocument` resolves the path, rejects traversal and escaping links, verifies the extension, and opens through a sandboxed parser. Never expose a generic shell or filesystem capability.

## Protocol and File Handling

Parse custom URLs with a strict grammar. Reject nested credentials, oversized payloads, ambiguous percent-encoding, remote file URLs, and unsupported actions. Treat drag-and-drop and file-open events as attacker-controlled input.

## Signed Updates

Pin the update public key outside the downloaded bundle. Verify the manifest signature first, then the binary signature, version monotonicity, channel, target platform, and expected digest before installation. A verification failure aborts and records an alert; it never falls back to HTTP or an older signer.

## Sandboxing and Secrets

Request only required filesystem, network, JIT, camera, microphone, and automation entitlements. Store tokens in the OS credential vault, keep them out of renderer storage and logs, and minimize their lifetime in process memory.

## Short Example

A deep link requests `file:///etc/passwd`. The privileged handler compares the resolved path with the approved document roots, rejects the request, records the link hash, and never opens the path in a privileged parser.
