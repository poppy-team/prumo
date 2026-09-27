# MCP Tooling Specification — Harbor Notes Server (2.1.0)

## 1. Package and Protocol Pins
- **Package**: `harbor-notes-mcp` 2.1.0, changelog entry 2026-09-17.
- **Protocol**: `2025-06-18`; SDKs: TypeScript 1.12.0 (server), Python 1.9.0 (scripting client in CI).
- **Capabilities advertised**: tools, resources with subscribe, prompts; sampling templates v3, elicitation for approval flows.
- **Spec date and owner**: 2026-09-17, developer-experience pod (Nadia Ferreira).

## 2. Transport Contracts

| Transport | Use | Contract | Measured |
|---|---|---|---|
| stdio | local CLI and editor subprocess | 10 MB message cap, SIGTERM shutdown within 2 s | shutdown p95 1.1 s, oversize message rejected `-32600` |
| Streamable HTTP | remote Harbor agents | resumable session IDs in Redis, 60 s heartbeat, TLS via Harbor edge | reconnect inside window resumes session, 42 of 42 drills |

## 3. Session Behaviors

| Behavior | Setting | Result |
|---|---|---|
| Reconnect backoff | 200 ms base, 5 s cap, 8 attempts, jitter | no reconnect storm in chaos test (50 drops) |
| Request timeout | 30 s typed `Timeout` | 0 silent hangs in load run |
| Progress | every 5 s on tools over 10 s | cancel propagates within 1 s, GPU work halted |
| Idempotency | client `callId` deduped 24 h | retried `note_publish` executed once in kill drill |

## 4. Conformance and Interop

| Battery | Result |
|---|---|
| Protocol battery v3.2, stdio | 214 of 214 pass |
| Protocol battery v3.2, HTTP | 213 of 214 pass, 1 known failure (upstream SDK-4478, chunked-trailers edge) |
| Harbor Desktop 3.1 | 100% interop pass (38 flows) |
| VS Code extension 0.24 | 97% (1 failure: elicitation theming, client bug VSC-9912, de-listed for themed approvals) |
| CI runner 0.8 | 100% interop pass |

## 5. Regression Evidence
- [x] Battery transcripts archived under `mcp/evidence/2.1.0/` for both transports.
- [x] Session-resume drill log: mid-`note_publish` kill, reconnect, exactly-once confirmed via `callId` ledger.
- [x] Published package with pins and changelog; known failure linked to SDK-4478.
- [x] `scripts/verify.sh` exits 0 on the skill package; tooling gate green on commit 91c4d0.
