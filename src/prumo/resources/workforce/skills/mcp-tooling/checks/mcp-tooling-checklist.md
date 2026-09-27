# MCP Tooling & Integrations — Verification Checklist

## 1. SDK & Protocol Baseline
- [ ] Server built on the official SDK with pinned versions (protocol `2025-06-18`, TS SDK 1.12.0 or Python SDK 1.9.0); lockfile committed.
- [ ] No hand-rolled JSON-RPC framing; all wire code paths go through SDK primitives.
- [ ] `initialize` advertises protocol version and capability flags; mismatched majors rejected with error `-32099` plus upgrade hint.
- [ ] Negotiated version logged per session for interop forensics.

## 2. Transports: stdio & Streamable HTTP
- [ ] stdio transport enforces the 10 MB message cap and shuts down cleanly on SIGTERM within 2 s.
- [ ] Streamable HTTP issues resumable server-side session IDs with a 60 s heartbeat; reconnect resumes the same session.
- [ ] TLS terminates correctly on remote endpoints; stdio servers never bind network sockets.
- [ ] Each advertised transport passes the full conformance battery independently.

## 3. Client Sessions, Timeouts & Progress
- [ ] Reconnect uses exponential backoff (200 ms base, 5 s cap, 8 attempts) with jitter.
- [ ] Every request carries a 30 s timeout producing typed `Timeout` errors, never silent hangs.
- [ ] Long tools emit progress notifications at least every 5 s; client cancellation propagates to the running tool.
- [ ] Sampling and elicitation flows validated against at least one real client, not just mocks.

## 4. Conformance, Interop & Release Evidence
- [ ] Protocol battery v3.2 (214 cases) green on all advertised transports; failures documented with transcripts.
- [ ] Interop matrix across declared clients (Harbor Desktop 3.1, VS Code ext 0.24, CI runner 0.8) recorded with pass rates.
- [ ] Session-resume drill logged: mid-call kill and reconnect shows exactly-once visible semantics.
- [ ] Package published with version, protocol/SDK pins, and changelog; de-listed clients noted with sign-off.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
