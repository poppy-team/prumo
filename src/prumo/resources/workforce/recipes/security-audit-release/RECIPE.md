# Security Audit & Release Gate

## Purpose
Execute comprehensive secure-by-design verification covering threat models, authorization matrices, multi-tenant isolation, memory safety sanitizers, and supply chain attestations.

## Preconditions
- System architecture and trust boundaries documented
- Candidate release build ready for gate verification

## Participating Agents
- `security-architect`
- `security-reviewer`
- `isolation-auditor`
- `release-verifier`

## Required Skills
- `security-threat-model`
- `security-authz-matrix`
- `security-saas-isolation`
- `memory-safety-sanitizers`
- `binary-security-mitigations`
- `supply-chain-security`
- `secrets-security`

## Quality Gates
- `threat-model-approved`
- `authz-matrix-clean`
- `isolation-verified`
- `sanitizers-clean`
- `supply-chain-passed`
- `release-signoff`

## Stop Conditions
- Zero critical or high unmitigated vulnerabilities; signed release manifest produced
