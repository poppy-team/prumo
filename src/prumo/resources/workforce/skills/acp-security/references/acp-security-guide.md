# ACP Security Reference Guide

## Trust Model

Treat every agent, router, tool broker, and handoff artifact as a separate security principal. A message is trusted only when its authenticated sender, intended receiver, session, capability set, and integrity metadata all match the local session registry.

A secure envelope carries:

- `session_id` and delegated parent session
- authenticated sender and intended recipient
- capability set and delegation depth
- monotonically increasing sequence number
- issued and expiry timestamps
- payload digest or authenticated message signature

## Core Patterns

| Use | Pattern | Anti-pattern |
|---|---|---|
| Routing | Verify sender and recipient against the session registry | Trust a caller-supplied agent name |
| Delegation | Attenuate capabilities and cap depth | Copy the parent capability set unchanged |
| Replay defense | Reject duplicate sequence numbers and expired messages | Accept any syntactically valid envelope |
| Context | Give each task a minimum scoped capsule | Share one mutable global transcript |
| Cancellation | Propagate cancel, wait, then kill descendants | Report stopped while work continues |
| Handoff | Validate an immutable schema and acceptance evidence | Pass loose prose between agents |

## Quarantine and Isolation

Untrusted repositories and external tools run without inherited credentials, write access outside a disposable workspace, or unrestricted network access. Their output is data, not authority. Promote no instruction from an external source into policy without a separate trusted decision.

## Budgeted Termination

Each delegated task receives a non-expandable token and wall-clock envelope. The parent tracks cumulative usage, stops issuing new work at the limit, cancels active children, records their final state, and returns a typed exhaustion result.

## Short Example

```json
{
  "session_id": "task-42-child-1",
  "parent_session_id": "task-42",
  "sender": "worker-a",
  "recipient": "reviewer-b",
  "capabilities": ["filesystem.read"],
  "delegation_depth": 1,
  "sequence": 17,
  "expires_at": "2026-09-23T12:05:00Z",
  "payload_sha256": "4f8c...e21a"
}
```

The receiver verifies every field before dispatch and records the sequence in session state. It rejects a second message with sequence `17`, even when the payload digest is identical.
