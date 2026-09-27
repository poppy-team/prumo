# Codex Integration

This guide explains how to integrate Prumo with Codex agents.

## Compilation Output

Running `prumo-agent compile codex` produces an integration layer tailored for Codex:

```text
.codex/
├── AGENTS.md               # Primary entrypoint for Codex
└── skills/
    └── <skill_id>/         # Fully resolved skill packages (copied over)
```

## How Codex Consumes Compiled Skills

Codex agents are instructed via `AGENTS.md` to look in the `.codex/skills/` directory for available tools and instructions. The compiler ensures all templates, scripts, and checks are bundled correctly.

## Step-by-Step

1. `prumo-agent init`: Initialize the project configuration.
2. `prumo-agent compile codex`: Generate the Codex integration artifacts.
3. Verify output: Check `.codex/AGENTS.md` and ensure skills are present.

## Customization

You can override skills locally by placing modified versions in `src/prumo/resources/custom_skills/` before compiling.

## Troubleshooting

- **Missing Skills:** Check your project's `stack` and `features` configuration to ensure the skills match the resolver's select criteria.
- **Malformed AGENTS.md:** Ensure your skill manifests are valid JSON.
