# Agent Client Protocol (ACP) Transport & Multi-Agent Security Checklist

- [ ] Agent message sender identities cryptographically verified on all channels
- [ ] Delegation depth restricted (max 1 or 2); subagents inherit reduced capabilities
- [ ] Message integrity protected with timestamps, nonces, and SHA-256 checksums
- [ ] Context memory strictly isolated; minimal Working Context Capsules used
- [ ] Token budgets capped per agent task; auto-terminate upon budget exhaustion
- [ ] Deterministic cancellation handling kills child tasks and processes cleanly
- [ ] Task handoffs pass immutable state snapshots meeting formal schema contracts
