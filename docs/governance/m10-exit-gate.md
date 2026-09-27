# M10 OpenCode Native Harness — Exit Gate

Status: **COMPLETE (M10 Exit Gate Passed)**

## Exit gate criteria

Milestone M10 exit gate (per `docs/development/phases.md:365`):
`prumo-agent connector install opencode` produces a fully functional native integration.

---

### 1. OpenCode Native Compiler

- Implemented in `internal/connectors/opencode/opencode.go` and integrated into `internal/cliops/compile.go`.
- Invoked via:
  - `prumo-agent compile opencode` or `prumo-agent compile --target opencode`
  - `prumo-agent connector install opencode` (and backward-compatible `prumo-agent install connector opencode`)
- Compiles a complete `.opencode/` workspace:
  - `.opencode/opencode.json`: workspace configuration adhering to OpenCode schema.
  - `.opencode/plugins/prumo.ts`: native TypeScript plugin providing tool guards and session lifecycle hooks.
  - `.opencode/agents/prumo.md`: primary Prumo orchestrator agent.
  - `.opencode/agents/architect.md`, `executor.md`, `verifier.md`: specialized subagents.
  - `.opencode/skills/`: mapped domain skills (`code-review`, `goal-management`, `evidence-collection`).
  - `.opencode/commands/prumo.json`: registered slash commands (`/goal`, `/plan`, `/trace`, `/experience`, `/adopt`, `/status`).
  - `.opencode/guards/tool-policy.json`: tool guard security policies.
  - `.opencode/.prumo-generated.json`: machine-readable ownership marker with version and managed flags.

---

### 2. Tool Guards (Pre-Tool Validation)

- Invariant: Tool executions pass through synchronous veto (`pre_tool_block`).
- Prohibits destructive shell commands (`rm -rf /`, `mkfs`, fork bombs).
- Protects sensitive paths (`.git/`, `.prumo/credentials`, `.env`) from destructive writes.
- Flags high-risk mutations (`git push --force`, `git reset --hard`) requiring explicit confirmation.

---

### 3. Session Lifecycle Hooks

- `session.start`: Injects Lean Progressive Context (LPC) and active Goal details without conversational transcript bloat.
- `session.end`: Emits structured session completion events into `.prumo/experience/`.
- `tool.before_execute` / `tool.after_execute`: Intercepts and logs tool telemetry deterministically.

---

### 4. Connector Contract & Cleanup Integrity

- Conforms to `schemas/connector-contract.schema.json`.
- Enforces strict enforcement and supports standard capabilities (`advise`, `restrict_tools`, `pre_tool_block`, `post_tool_verify`, `isolate_subagents`, `session_hooks`, `native_plugin`, `commands`, `subagents`).
- Generates `CleanupManifest` in `PRUMO_HOME/connectors/opencode/cleanup.json`.
- Supports atomic and safe cleanup via `prumo-agent connector uninstall opencode`, leaving user files intact.

---

### 5. Verification Evidence

- Unit and integration tests in `internal/connectors/opencode/opencode_test.go`:
  - `TestOpenCodeContract`: verifies capability set, enforcement mode, and lifecycle hooks.
  - `TestOpenCodeCompileAndValidate`: verifies directory structure, plugin code, and ownership marker.
  - `TestOpenCodeInstallAndUninstall`: verifies manifest registration, cleanup manifest creation, and safe uninstallation.
- CLI tests in `cmd/prumo/connector_commands_test.go`:
  - Validates `prumo-agent connector list`, `prumo-agent connector install`, `prumo-agent connector validate`, `prumo-agent connector uninstall`, and `prumo-agent install connector`.
- All tests pass with `-race` enabled.
