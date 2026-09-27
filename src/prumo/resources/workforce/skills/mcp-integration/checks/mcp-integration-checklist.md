# MCP Integration — Verification Checklist

## 1. Tool Surface & Naming
- [ ] Every exposed tool has a verb-first semantic name (`issue_search`); no `execute`, `run`, or `eval` tools on shared servers.
- [ ] Tool count justified against the candidate inventory; excluded operations documented with reasons.
- [ ] Resources use stable URIs with pagination cursors and ETags; no unbounded list responses (page cap at or below 50 items).
- [ ] Tool descriptions are injection-hardened: no imperative instructions to the client model, no hidden prompt content.

## 2. Schemas & Error Contracts
- [ ] All tool inputs declare strict JSON Schema with `additionalProperties: false`; unknown fields reject with typed errors.
- [ ] Outputs declare schemas; large payloads paginate instead of truncating silently.
- [ ] Error envelopes typed (`code`, `message`, `data`); 40-case probe battery shows zero stack traces, SQL, or internal hostnames.
- [ ] Version negotiation handled: server advertises protocol `2025-06-18` and rejects incompatible clients with a clear error.

## 3. Authority, Consent & Scoping
- [ ] Each tool maps to minimal OAuth scopes; read tools never carry write scopes.
- [ ] Destructive tools require `confirm: true` plus client-side human approval policy.
- [ ] Per-call audit log present: tool name, principal, args hash, timestamp; sample retained 90 days.
- [ ] Rate limits enforced per tool and principal (e.g., 200 calls/min); over-limit returns typed throttling errors.

## 4. Performance & Regression Evidence
- [ ] Load test (200 calls/min, 15 min) holds p95 latency under 800 ms with zero scope violations.
- [ ] Tool catalog published with examples, limits, and changelog for the release.
- [ ] Integration tests pin every schema against the SDK validator on each merge.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
