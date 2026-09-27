# OpenCode Native Integration

This guide details the Prumo native harness integration for [OpenCode](https://opencode.ai).

## Overview

Prumo integrates natively with OpenCode via the OpenCode Native Harness (`opencode`), providing:
- In-process TypeScript plugin (`.opencode/plugins/prumo.ts`)
- Primary Prumo orchestrator agent (`.opencode/agents/prumo.md`)
- Domain-specific subagents (`architect`, `executor`, `verifier`)
- Pre-tool validation (tool guards) with synchronous veto (`pre_tool_block`)
- Lifecycle session hooks (`session.start`, `session.end`, `tool.before_execute`, `tool.after_execute`)
- OpenCode slash commands (`/goal`, `/plan`, `/trace`, `/experience`, `/adopt`, `/status`)
- Cleanup manifests for zero-residue uninstallation

## Installation

Install the OpenCode native harness into your repository:

```bash
prumo-agent connector install opencode
# or via backward-compatible command
prumo-agent install connector opencode
```

This compiles the `.opencode/` workspace directory:
```text
.opencode/
├── opencode.json             # OpenCode workspace configuration
├── plugins/
│   └── prumo.ts              # TypeScript plugin (tool guards & lifecycle hooks)
├── agents/
│   ├── prumo.md              # Primary Prumo orchestrator
│   ├── architect.md          # Architecture & schema subagent
│   ├── executor.md           # Implementation subagent
│   └── verifier.md           # Quality gate & test subagent
├── skills/
│   ├── code-review/SKILL.md
│   ├── goal-management/SKILL.md
│   └── evidence-collection/SKILL.md
├── guards/
│   └── tool-policy.json      # Tool guard security policies
├── commands/
│   └── prumo.json            # Slash commands
└── .prumo-generated.json     # Ownership marker
```

## Tool Guards

Tool guards enforce the Prumo trust model prior to tool invocation (`pre_tool_block`):
- Dangerous shell commands (e.g. `rm -rf /`, `mkfs`) are blocked immediately.
- Sensitive files (`.git/`, `.prumo/credentials`, `.env`) are protected from destructive write operations.
- Operations requiring human confirmation (e.g. `git push --force`) trigger user review.

## Lifecycle Hooks

- **`session.start`**: Injects Lean Progressive Context (LPC) and active Goal details without bloating initial context.
- **`session.end`**: Synthesizes session summaries and emits structured session events into `.prumo/experience/`.
- **`tool.before_execute` / `tool.after_execute`**: Records tool calls and results deterministically for auditability.

## Validation and Cleanup

Validate the native harness installation:
```bash
prumo-agent connector validate opencode
```

Safely uninstall without deleting repository files:
```bash
prumo-agent connector uninstall opencode
```
