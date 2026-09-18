# ADR 003: Language Engineering & Safety Skills Subsystem

## Status

Accepted

## Context

Prumo manages engineering workflows and guides AI coding agents across multiple platforms and technologies. Previously, skills provided high-level procedural guidelines for tasks like clean code, accessibility, and documentation. However, multi-language systems programming requires non-negotiable, machine-executable implementation contracts rather than generic textual advice. Low-level systems languages (C++, C, Assembly, C3, D, Zig, Odin) possess distinct memory, lifetime, concurrency, and safety models. Without enforceable contracts, AI code generators are prone to memory unsafety, naked heap allocations, raw pointer arithmetic, type aliasing violations, and silent error swallowing.

## Decision

We establish the **Language Engineering & Safety Skills Subsystem** within the Prumo, categorized across four families:
1. **Systems & Low-Level**: C++ Modern (C++20/C++23 with paranoid safety), ISO C (bounded buffers), Assembly x86/x86-64 (ABI compliance), C3, Dlang, Zig, and Odin.
2. **Managed & Modern Compiled Safe**: Rust, Go, Swift, C#, Java, Kotlin, Dart, Elixir.
3. **Dynamic & Scripting**: TypeScript, JavaScript, Python, Ruby, PHP, Lua, Bash.
4. **Domain & DSL Engineering**: HTML5, CSS, SQL, GraphQL, Shaders (GLSL/HLSL/WGSL), Dockerfile, Terraform/HCL, YAML, JSON.

Key architectural mechanisms:
- **Executable Contracts**: Each skill follows Lean Progressive Context (`SKILL.md` bounded core, `checks/`, `templates/`, `references/`, `scripts/`) and registers machine selectors, required capabilities, and required evidence in `skills.json` under schema_version 2.
- **Escape-Hatch Registry**: Banned constructs (e.g. `reinterpret_cast`, raw `malloc`/`free`, `(void*)`, `@ptrCast`, `@trusted`, suppressions) cannot be introduced without formal registration in `.prumo/escape-hatches.json` or inline annotations with explicit safety invariants and quarantine boundaries (`schemas/escape-hatch.schema.json`).
- **Native Go Tooling**: Embedded `prumo-agent tool check-escape-hatches` scanner provides instant verification in CLI, pre-commit hooks, and CI pipelines.
- **Adaptive Quality Gates & Evidence**: Automated generation of `evidence.json` matching `schemas/evidence.schema.json` verifying compiler warnings, linters, and sanitizers (ASan, UBSan, TSan, Valgrind).
- **Connector Compilation**: Host connectors (Antigravity, OpenCode, Codex, Claude Code, Gemini) emit language skills, rules, and audit hooks directly into target harnesses.

## Consequences

Positive:
- AI agents are bound to strict, test-driven, verifiable implementation loops.
- Memory safety, bounds verification, and lifetime rules are enforced deterministically by toolchains and scanners.
- Clear separation between safe code and quarantined low-level escape hatches.
- Seamless stack-based workforce resolution via `internal/resolver`.
- Zero additional external dependencies; verification tooling runs natively in pure Go.

Negative:
- Codebases with legacy idioms require formal escape-hatch registration or ratcheting baselines before passing quality gates.

## References

- `schemas/escape-hatch.schema.json`
- `schemas/language-profile.schema.json`
- `src/prumo/resources/catalog/skills.json`
- `internal/tooling/escape_hatch.go`
- `docs/architecture/overview.md`
