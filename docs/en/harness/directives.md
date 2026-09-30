# Directives & Task DAGs

In Prumo, AI agents do not work from open-ended, vague instructions. Instead, the Prumo compiler turns Goals and Tasks into structured **Executable Directives** in JSON Schema Draft 2020-12 format.

## Structure of a Directive

Each directive encapsulates the strictly necessary context (LPC) and the corresponding validation rule:

```json
{
  "$schema": "https://prumo.dev/schemas/directive.json",
  "directive_id": "DIR-P01-G01-T01",
  "goal_ref": "P01-G01",
  "task_ref": "T01",
  "actor": "systems-core",
  "context_capsule": {
    "target_files": ["internal/auth/jwt.go", "internal/auth/jwt_test.go"],
    "referenced_contracts": ["security.trust-model", "architecture.dependency-rules"],
    "budget_tokens": 8000
  },
  "acceptance_criteria": [
    "TestJWTVerification must pass with at least 90% coverage",
    "No external package beyond the stdlib may be added to go.mod"
  ],
  "verification_command": "go test ./internal/auth/... -race"
}
```

## Directed Acyclic Graph (DAG) of Tasks

When a Plan is formulated, its tasks form a deterministic DAG:

```mermaid
graph TD
  T01["T01: JWT Structure and Types"] --> T02["T02: HMAC Signing Algorithm"]
  T01 --> T03["T03: Claims Parser and Time Validation"]
  T02 --> T04["T04: Test Suite and Conformance Gate"]
  T03 --> T04
```

1. **Safe Parallel Execution**: Tasks with no mutual dependencies can be executed simultaneously by distinct specialist agents.
2. **Blocking Gates**: Advancing to the next task is only released once the previous gate's success evidence has been produced.
3. **Deterministic Rollback**: If a verification fails, Prumo preserves the intermediate state and creates a diagnostic checkpoint without corrupting the main Git tree.
