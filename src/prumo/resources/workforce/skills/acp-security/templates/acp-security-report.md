# ACP Security Audit — Research Worker Session

## Scope
- **Component**: `research-worker` ACP session broker
- **Review date**: 2026-09-23
- **Assessor**: identity-security-agent
- **Entry points**: task delegation, result handoff, cancellation, token exhaustion

## Trust Contract
- Parent session may delegate `filesystem.read` and one network domain.
- Child capabilities are read-only and delegation depth is capped at one.
- Envelope signature, session IDs, sequence numbers, and expiry are mandatory.

## Finding ACP-01
- **Severity**: High
- **Observed**: A replayed result with sequence `17` was accepted after the original result.
- **Impact**: A malicious peer could repeat a completed tool result.
- **Remediation**: Persist the highest accepted sequence per session and reject duplicates before dispatch.
- **Regression test**: Submit the same signed envelope twice; only the first may reach the handler.

## Finding ACP-02
- **Severity**: Medium
- **Observed**: Cancellation stopped the child but left its artifact download running.
- **Remediation**: Propagate cancellation to descendants, await bounded shutdown, then terminate the process group.
- **Regression test**: Cancel during download and assert no process or network lease remains.

## Release Decision
Blocked until ACP-01 is fixed and the replay, isolation, and deterministic-cancellation suites pass.
