# GEMINI.md

This project uses Prumo v0.4 with Google Antigravity.

- Read `ENTRYPOINT.md`, `prumo.json`, `docs/PRUMO.md`, and the active Goal before taking action.
- Treat Prumo as an external CLI utility available in PATH (`prumo`). Run `prumo <command>` or `prumo --help` for project operations and lifecycle. Do not inspect or search for internal framework development source code.
- Use Lean Progressive Context: smallest sufficient context, progressive expansion, pointer over payload.
- Do not scan the entire codebase unless explicitly requested.
- Implement only locked Goal scope; test evidence determines completion.
- Test-Driven Implementation: Every implementation requires automated tests (unit, integration, conformance).
- Security First: Zero hardcoded secrets, follow least privilege, audit dependencies and sanitize inputs.
- Clean Architecture: Modular design, high cohesion, low coupling, no circular dependencies.
- Update documentation and `CHANGELOG.md` alongside code changes.
- Ensure every directory in the project contains an explanatory `README.md`.
