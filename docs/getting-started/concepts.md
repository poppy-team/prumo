# Core Concepts

Prumo is a Go-based, Git-native protocol and CLI for engineering software with humans and AI agents.

- **Repository over conversation memory:** canonical project knowledge lives in Git.
- **Protocol over harness:** OpenCode, Codex, Claude Code, Gemini and other clients are adapters.
- **Canonical before generated:** Markdown, JSON, JSON Schema and Goals are canonical; adapters, caches, indexes and runtime context are derived.
- **Goals as outcome unit:** acceptance criteria become integrity-locked after `LOCKED`.
- **Plans and Tasks:** Plans form acyclic dependency graphs (DAGs).
- **Workforce packages:** Agents, Skills and Recipes are versioned, resolvable packages.
- **Lean Progressive Context:** select the smallest sufficient context and expand only with evidence.
- **Evidence over assertion:** completion requires tests, builds, reviews, scans or other declared evidence.
- **Deterministic before probabilistic:** protocol invariants and gates do not depend on LLM behavior.
- **Python as oracle:** Python v0.3 remains for compatibility testing during the migration, not normal v0.4 runtime use.
