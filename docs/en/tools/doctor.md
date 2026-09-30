# Repository Diagnostics (`prumo doctor`)

The `prumo doctor` command is the repository's physician. It performs a thorough, multi-level inspection to ensure the project is sound and ready for safe autonomous execution.

## Check Categories

1. **Goal & Plan Integrity**:
   - Verifies that all Goals in the `LOCKED` state have valid SHA-256 digests and were not manually modified in Git.
   - Validates that the task DAG of active Plans has no cycles or orphan nodes.

2. **JSON Schema Conformance**:
   - Validates `prumo.json`, profiles, and manifests against Draft 2020-12 of the canonical JSON Schemas embedded in the binary.

3. **Environment & Tooling**:
   - Detects the presence of Git and the integrity of the local repository.
   - Checks the executables required by active profiles and recipes (compilers, linters, test runners).

4. **Connectors & Adapters**:
   - Verifies that the adapters compiled into `CLAUDE.md`, `.cursorrules`, etc., are in sync with the latest definitions in `prumo.json`.

## Example Run

```bash
prumo doctor
```

Typical output in text mode:

```text
[OK] Git repository clean and initialized
[OK] prumo.json valid against schema v1
[OK] 14 Goals verified, SHA-256 locks intact
[OK] Task DAGs verified without cycles
[OK] Adapters synchronized: claude-code, generic
Prumo repository health check passed (100% operational).
```
