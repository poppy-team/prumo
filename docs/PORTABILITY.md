# Portability and Platform Adapters

The generic Markdown protocol is the compatibility floor. Platform adapters improve ergonomics but cannot own essential policy.

Supported initial targets:

- `generic`
- `chatgpt`
- `claude`
- `kimi`
- `codex`
- `claude-code`
- `traycer`

## Generated adapter rule

Platform output is generated and replaceable.

Codex/Claude Code may require platform-native role/skill files. Generic/chat targets should use temporary compiled entrypoints/context under `.prumo/runtime/compiled/`, not committed duplicate context packs.

Adapters should point agents back to:

1. `ENTRYPOINT.md`;
2. `prumo.json`;
3. `docs/PRUMO.md`;
4. active Goal;
5. the Progressive Context rules.

## Context rendering

A platform/model adapter may render Context IR as compact KV, minimal Markdown or lightweight tags when benchmarked. This does not change canonical storage.

## Persistent format portability

Canonical project formats:

- Markdown;
- JSON.

SQLite is runtime only.

YAML is legacy import/read compatibility.

## Conversation commands

Portable conventions:

- `Prumo: bootstrap`
- `Prumo: finalize`
- `Prumo: update`
- `Prumo: audit`
- `Prumo: recover`
- `Prumo: migrate`

They describe intent and are not required platform syntax.
