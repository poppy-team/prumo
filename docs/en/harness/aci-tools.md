# ACI Tooling & Sandboxing

The **ACI (Agent-Computer Interface)** of the Prumo Harness is the set of precision tools provided to agents during directive execution.

## ACI Design Principles

Unlike naive approaches that hand the model an unrestricted shell, Prumo implements fundamental safeguards:

1. **Hard Output Limits (Bounded Output)**: Commands run by the agent have strict limits on the bytes captured from stdout/stderr to prevent context pollution and token overflow.
2. **Non-Negotiable Timeouts**: Every child process has a time limit (default 30 seconds, configurable per directive) before it is safely terminated.
3. **Scope Isolation (Working Directory Sandboxing)**: File write operations can only reach targets inside the root allowed by the workspace.
4. **Native Static Analysis (AST Tools)**: The agent can inspect syntax nodes without loading huge files into memory.

## Tools Available Out of the Box

| Tool | Identifier | Function |
|---|---|---|
| **Safe Execution** | `execute_process` | Launches binaries with a timeout, controlled environment variables, and bounded log capture. |
| **Atomic Editing** | `replace_file_content` | Contiguous replacement of text blocks with prior validation of a unique match. |
| **File Creation** | `write_to_file` | Writes new artifacts, with an option to automatically create parent directories. |
| **Bounded Reading** | `view_file` | Reads paginated slices of code (1-indexed slicing) with a byte bound. |
| **AST Inspection** | `ast_inspect` | Semantic search for functions, structs, interfaces, and type declarations. |
| **Gate Verification** | `gate_eval` | Evaluation of test commands and deterministic conformance suites. |

## Sandboxing Definition Example

Execution policies are declared in the project or profile configuration file (`prumo.json`):

```json
{
  "harness": {
    "sandbox": {
      "mode": "restricted",
      "allowed_commands": ["go", "git", "cargo", "pnpm", "make"],
      "max_timeout_seconds": 60,
      "max_output_bytes": 65536,
      "deny_network_for_tests": true
    }
  }
}
```
