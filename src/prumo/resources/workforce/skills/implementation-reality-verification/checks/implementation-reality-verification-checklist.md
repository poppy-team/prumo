# Implementation Reality Verification Checklist

## Claim Inventory
- [ ] The exact user-visible or operational claim is stated before testing.
- [ ] Entry points, call sites, dependency injection, routes, flags, and lifecycle hooks are inventoried.
- [ ] Code with no inbound path is classified as dead, experimental, or incorrectly wired.

## Reachability
- [ ] A concrete path is traced from a real entry point to the claimed behavior.
- [ ] Bypasses, feature flags, error branches, and alternative implementations are included.
- [ ] Presence in a source file is not used as reachability evidence.

## Exercise and Oracle
- [ ] Tests traverse the claimed branches with representative inputs.
- [ ] Assertions can fail and expected results are explicit.
- [ ] Errors, retries, cancellation, persistence failure, and partial state are exercised where relevant.

## False-Green Audit
- [ ] Mocks do not replace the capability under test in claimed integration evidence.
- [ ] Swallowed errors, weak fixtures, unconditional success, and skipped tests are reviewed.
- [ ] Critical-path mutations or equivalent fault injection cause the oracle to fail.

## Terminal State
- [ ] Implemented, reachable, exercised, evidenced, verified, accepted, and released are tracked separately.
- [ ] Completion claims include exact commands, output summaries, and residual risk.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
