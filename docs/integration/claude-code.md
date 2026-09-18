# Claude Code Integration

This guide explains how to integrate Prumo with Claude Code.

## Compilation Output

Running `prumo-agent compile claude-code` produces:

```text
.claude/
├── CLAUDE.md               # Primary entrypoint for Claude
└── skills/
    └── <skill_id>/         # Fully resolved skill packages
```

## How Claude Code Consumes Compiled Skills

Claude is directed by `CLAUDE.md` to utilize the specific directory structure of `.claude/skills/`. The compiler formats the instructions to align with Claude's expected prompt structures.

## Step-by-Step

1. `prumo-agent init`: Initialize Prumo.
2. `prumo-agent compile claude-code`: Build the Claude integration.
3. Verify output: Review `CLAUDE.md`.

## Customization and Overrides

Use project-local overrides in your Prumo config to modify skill selection or agent personas specifically for the Claude compilation target.
