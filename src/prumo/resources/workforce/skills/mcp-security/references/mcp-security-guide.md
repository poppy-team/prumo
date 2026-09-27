# MCP Security Reference Guide

## Tool Gateway

Place every tool behind a gateway that authenticates the client, authorizes the exact tool and arguments, enforces policy, records the decision, applies resource limits, and returns a bounded result. Do not expose a raw shell, unrestricted filesystem, or ambient credential path as a tool.

## Schema and Capability Enforcement

Validate inputs against the advertised JSON Schema with additional properties rejected. Enforce string, array, numeric, and byte bounds before allocation or I/O. Capabilities are declared, scoped, expiring, and attenuated at handoff.

## Untrusted Output and Prompt Injection

Files, web pages, database records, and tool errors are untrusted data. Preserve their boundaries in the client, reject embedded authority claims, and require a separate trusted policy decision before any instruction or side effect follows.

## Filesystem and Process Isolation

Resolve paths, deny protected roots, and apply operation-specific access. External servers receive a sanitized environment, low-privilege identity, disposable workspace, restricted network, timeout, and output cap. Credentials stay in the gateway and never become tool arguments.

## Side Effects and Approval

Classify tools as read-only or mutating. Journal intent before a mutation and outcome afterward with sanitized parameters and result hashes. Deletion, migration, publication, credential changes, and policy changes require explicit human approval bound to the reviewed action.

## Patterns and Anti-Patterns

| Do | Avoid |
|---|---|
| Authorize the concrete tool call | Trust a client's broad server capability |
| Validate before allocating or I/O | Truncate oversized input after parsing |
| Mark tool output as untrusted data | Paste retrieved text into system instructions |
| Bind approval to exact arguments | Ask for approval once for a future action |
| Journal sanitized intent and outcome | Log secrets and complete file contents |

## Short Example

A request to read `.env` resolves under the repository root but matches a protected-path rule. The gateway returns `E_ACCESS_DENIED`, records the actor, tool, path hash, and policy version, and performs no file open.
