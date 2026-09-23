# ADR 017 — Subagent delegation is one level, fixed, and constructed rather than configured

Status: Accepted (2026-09-23)
Relates to: GAP-003, GAP-022, GAP-101, GAP-109, GAP-167, GAP-168

## Context

The workforce agent manifests carry `may_delegate` and `max_delegation_depth`
(`schemas/agent.schema.json:27-37`). The framework invariant in `FRAMEWORK.md:24`
says deep recursive LLM execution is not a core requirement and that single-level
isolated delegation is the default maximum unless a project explicitly enables
and benchmarks an experiment.

None of that exists as code. `may_delegate` has no Go reference. `max_delegation_depth`
is only read while generating configuration (`internal/app/goalplan.go:120-123`,
`internal/protocol/goals/goals.go:92`, `internal/cliops/ops.go:94-102`). All 26
agent manifests set `may_delegate: false`, so the framework currently has a
depth field whose legal values are all "no delegation".

The primitives exist and are tested — `harness/team.Runner`, `RunWork`,
`AllocateWorktree`, `MergeWorkspaces`, `ChildHandoff` — but have no production
caller. `harness/eval/eval_test.go:24` is the only importer.

The comparable products split on this axis:

- **opencode** makes the relationship structural: an agent has
  `mode: primary | subagent | all`. A `subagent` cannot be the session's main
  agent, and the built-in `general` subagent "cannot launch more subagents".
- **Claude Code** allows depth but fixes the ceiling: depth is counted as the
  number of subagent levels below the main conversation, an agent at depth five
  does not receive the `Agent` tool, and the limit is explicitly *not
  configurable*. It also fixes the depth of a background subagent at spawn time
  so resuming cannot extend it.

## Decision

1. **One level, fixed.** A subagent may delegate further exactly zero times.
   This is a property of the child runner's construction, not a policy value read
   at runtime, so there is no configuration that can widen it.

2. **The depth field stays, and becomes descriptive rather than prescriptive.**
   `max_delegation_depth` remains in the agent manifest and is now required to be
   `0` for every agent that the framework can launch as a child. A project that
   wants a different value must propose an ADR and, per `FRAMEWORK.md:24`, a
   benchmark. A linter rejects a manifest with a value other than `0`.

3. **`may_delegate` becomes enforced, not decorative.** The delegation tool is
   constructed with a child factory that has no reference back to the delegation
   entry point. `agent.delegate` exists on the parent; the child has no such
   tool in its catalog.

4. **Context isolation is the contract.** The child starts a fresh conversation
   and receives only the task prompt the parent wrote. Tool calls, intermediate
   reasoning and file contents stay inside the child. Only its final message,
   plus its evidence references and checkpoint pointer, return to the parent.
   This is the property that makes delegation worth doing at all.

5. **A worktree is a git boundary, not a security boundary.** `isolation: worktree`
   is declared as what it is. Filesystem tools are confined to the worktree
   root; `process.exec` is confined by the container runtime that the parent run
   already uses, not by the working directory. A child cannot read or write
   outside its worktree through any tool.

6. **A yielded or cancelled child is not a successful child.** `PhaseYield`,
   cancellation and a lost run are failures for the parent's accounting
   purposes. A child that produced no result is not "done".

7. **Delegation terminates.** Suggested-delegation mode carries a hard cap on
   rounds and rejects a suggester that repeats a role, so a pathological
   suggester cannot loop or grow memory without bound.

## Consequences

- The framework gets working multi-agent execution without inheriting the
  coordination overhead, token cost and failure modes that make deeper trees
  unreliable. opencode and Claude Code both document diminishing returns past a
  handful of workers; one level captures nearly all of the value.
- A project that genuinely needs a tree pays for an ADR plus a benchmark, which
  is the intent of the `FRAMEWORK.md` invariant rather than a default.
- The delegation budget is inherited: the parent's remaining budget is the
  ceiling for the sum of its children, reserved before the child is spawned.
  This closes the current gap where each child tracker is independent
  (`internal/harness/team/bind.go:41-63`).
- `AllocateWorktree` and `MergeWorkspaces` get their first production caller.
  Merge-back runs the project's CI gate before the result is offered to the
  parent, so a child's work cannot silently land.
