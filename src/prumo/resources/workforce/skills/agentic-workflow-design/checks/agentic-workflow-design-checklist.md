# Agentic Workflow Design — Verification Checklist

## 1. Stages & Ownership
- [ ] Goal decomposed into named stages with exactly one owning role each.
- [ ] Each stage names the evidence it produces and the gate it must pass.
- [ ] No stage lists shared or ambiguous ownership.

## 2. Topology Selection
- [ ] Coordination topology named (pipeline, fan-out/fan-in, supervisor, blackboard).
- [ ] Written rationale connects the topology to the Goal's dependency structure.
- [ ] Fan-out shards define a merge stage with conflict-resolution rules.

## 3. Handoff Contracts
- [ ] Every inter-stage boundary has a versioned contract (inputs, outputs, schema version).
- [ ] Contracts reference artifact schemas stored alongside the recipe.
- [ ] Schema mismatch handling is defined (reject with reasons, never silent coerce).

## 4. Transitions, Idempotency & Caps
- [ ] Allowed state transitions enumerated; all other transitions forbidden.
- [ ] Retry counter capped at 2 per stage; delegation depth capped at 3.
- [ ] Every stage action verified safely re-runnable with no duplicate side effects.

## 5. Failure Analysis & Sign-off
- [ ] Walkthrough covers stage failure, reviewer rejection, and handoff mismatch, each terminating in done, escalated, or aborted.
- [ ] No silent-stall path remains; every path has an owner and a timeout.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
