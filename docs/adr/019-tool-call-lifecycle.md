# ADR 019 — One mandatory tool-call lifecycle, owned by a single boundary

Status: Accepted (2026-09-23)
Relates to: GAP-110, GAP-114, GAP-115, GAP-116, GAP-117, GAP-126, GAP-127, GAP-128, GAP-143, GAP-158, GAP-169, GAP-170

## Context

`docs/runtime/control-plane.md:74-88` specifies the intended tool lifecycle:

```
Agent → Tool Gateway → Policy → Permission → Budget → Sandbox → Tool/MCP
```

with descriptor-declared filesystem, egress, credential, timeout, output and
schema controls. The implementation does not have that path. It has pieces:

| Piece | Location | Production caller |
|---|---|---|
| Descriptor registry | `internal/toolgateway/gateway.go` | `tool list/inspect/evaluate` only |
| Permission engine | `internal/harness/perm` | yes |
| Native executor | `internal/harness/aci` | yes |
| Container layer | `internal/harness/aci/container.go` | partially |
| MCP adapter | `internal/harness/mcp` | yes |
| Directive firewall | `internal/harness/directive` | **tests only** |
| Side-effect journal (toolgateway) | `gateway.go:199-245` | **tests only** |
| Side-effect journal (checkpoint) | `checkpoint/checkpoint.go:157-214` | **tests only** |
| Side-effect journal (runtime) | `internal/runtime/journal` | **none** |
| Failure taxonomy | `internal/runtime/failure` | **none** |
| Environment/sandbox port | `internal/environment` | **none** |

Three side-effect engines exist and none is called. The registry advertises four
tools (`read_file`, `write_file`, `exec_command`, `git_commit`) that have no
dispatch case (`cmd/prumo/d2_commands.go:18-61` vs `144-374`). The root gateway
is not the execution gateway: agent execution constructs an `aci.Executor`
directly (`cmd/prumo/agent_commands.go:571-618`).

The consequence that matters most for an LLM agent is in the model adapters.
Native tools have no `Specs`, so real providers are never told what tools exist
(`aci/aci.go:17-45`, `daemon.go:405-430`). Anthropic declares `ToolCalls: true`
and omits `req.Tools` from the request body entirely
(`internal/harness/model/anthropic.go:37-41`, `116-118`). OpenAI serialises tool
definitions correctly but never accumulates streamed `tool_calls` by index, and
the tool result is appended as a generic `role:tool` message with no
`tool_call_id` (`model/model.go:275-315`, `392-445`;
`internal/harness/runtime/runtime.go:507-510`). Only the fake provider can
complete a tool loop.

`ToolResult.Error` and `ExitCode` are discarded — only `Output` becomes the
observation (`runtime.go:450-455`) — so a failed tool reaches the model as an
empty, successful-looking message.

## Decision

1. **`internal/harness/toolcall` is the one execution boundary.** Every tool
   invocation, from every source, goes through it: native, container, MCP, ACP
   and external agents. There is no second path.

2. **The lifecycle is mandatory and ordered:**
   `declare → validate → policy → permission → budget → journal intent →
   sandbox → execute (timeout, bounded output) → structured result → audit`.
   A step cannot be skipped; where a capability is absent (no container, no
   egress mediation) the lifecycle says so explicitly rather than proceeding as
   if it were enforced.

3. **`Tool` is an interface, not a name in a switch.**
   `Specs`, `Schema`, `Timeout`, `OutputMax`, `Kind`, `Execute`. Tool declaration
   is part of the port, which removes the optional interface assertion that let
   the daemon run with no tools declared at all
   (`runtime.go:29-31`, `cmd/prumo/agent_commands.go:248-259`).

4. **Results are structured, never a bare string.**
   A typed error taxonomy replaces `ExitCode int` + `Error string`:
   `permission_denied`, `policy_denied`, `timeout`, `exit_nonzero`,
   `output_truncated`, `side_effect_unknown`, `unavailable_runtime`,
   `validation_failed`. The model receives the taxonomy and the output; the
   audit receives the fingerprint, duration and output digest.

5. **Side effects are journaled before they happen.** `RecordIntent` moves from
   tests into the execution path. A record found in `pending` after a crash is
   an `unknown_side_effect` and is **not** replayed automatically — the current
   behaviour returns `true` and re-executes (`checkpoint.go:186-203`), which
   contradicts `docs/runtime/control-plane.md:38-42`.

6. **Exactly one side-effect implementation survives.** The three engines become
   one; the others are deleted, not left for a future refactor.

7. **The directive firewall is either in the path or removed.**
   `NewRunnerWithDirective` is production-wired and its envelope, capability,
   timeout and result-budget constraints are enforced, not just
   `CanMutatePath` (`runtime.go:430-449`).

8. **Bounds apply before materialisation.** `os.ReadFile` and `CombinedOutput`
   currently complete in full and are truncated afterwards
   (`aci/aci.go:137-142`, `188-192`). Execution uses bounded readers with a
   hard ceiling, and a truncated result names the artifact holding the rest.

9. **Turn identity advances.** `TurnID` is computed per model request
   (`runtime.go:84-91`, `297-303`), so a provider reusing a tool-call id cannot
   reuse a permission or idempotency identity across turns.

10. **MCP is initialised.** The stdio transport performs `initialize` and
    `notifications/initialized` before `tools/list`
    (`mcp/mcp.go:235-321`), and `Fanout.Specs` returns native **and** MCP tools
    rather than replacing one with the other (`mcp/tools.go:105-107`).

## Consequences

- Real providers can complete a tool loop, because the declarations exist and
  the result feedback is a valid provider exchange. This unblocks the
  `Anthropic.ToolCalls: true` claim that is currently false.
- The fake provider stops being the only thing that works. The conformance
  suite can assert the same trajectory across every provider.
- Four registered tools either gain a dispatch case or leave the registry. A
  tool in the registry with no implementation is a promise the model can act on.
- Bounded output changes tool behaviour observably: a caller that relied on
  receiving a truncated-as-success result now gets `output_truncated` and an
  artifact reference. That is the correct direction, and it is a breaking change
  for any consumer parsing the output as complete.
- Deleting two of the three side-effect engines reduces the surface where a
  partial fix could again leave an implementation unwired.
