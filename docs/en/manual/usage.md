# CLI Usage Manual

## Recommended Engineering Flow

```text
init (or adopt) → validate → doctor → Goal → context → compile → evidence → review
```

Prumo keeps the canonical truth in the Git repository. The harness executes actions deterministically; Prumo enforces the protocol, policies, Goals, evidence, and adapters.

---

## 1. Initialize a New Project (`prumo init`)

Prumo supports zero-configuration initialization with automatic stack detection:

```bash
# Zero-configuration (detects the language and creates a Protocol v3 workspace)
prumo init

# With specific presets
prumo init --preset web --name web-portal
prumo init --preset service --stack go
prumo init --preset cli --name my-tool

# Preview the inferred profile before applying
prumo init --print-profile
```

---

## 2. Adopt an Existing Repository (`prumo adopt`)

To turn any existing codebase into a Prumo-governed project:

```bash
# Audit and generate non-destructive proposals
prumo adopt

# Apply the migration and configure the Protocol v3 prumo.json
prumo adopt --apply

# Specific inspection subcommands
prumo adopt scan .       # Index files
prumo adopt facts .      # Extract technical facts
prumo adopt classify .   # Architectural classification
prumo adopt scaffold .   # Dry-run of the proposed changes
```

---

## 3. Validate and Diagnose

```bash
# Structural validation of Protocol v3 schemas and contracts
prumo validate

# Full health diagnosis of the project, locks, and connectors
prumo doctor

# Output in a structured JSON envelope for pipelines
prumo --json doctor
```

`validate` checks JSON schemas and the integrity of required files. `doctor` audits the integrity of Goals, versioning, DAGs, workforce integrity, and the AI tools in the environment.

---

## 4. Create and Lock Goals

```bash
# Create a new goal
prumo goal new P00-G01 "Foundation" \
  --phase P00 \
  --objective "Establish a tested project foundation."

# List the project's goals
prumo goal list

# Transition the goal's state
prumo goal state P00-G01 PLANNED
prumo goal state P00-G01 LOCKED
```

State lifecycle:

```text
DRAFT → PLANNED → LOCKED → EXECUTING → VERIFYING → REVIEWING → DONE
```

Locked goals (`LOCKED`) carry a SHA-256 cryptographic digest and require a formal amendment (`goal amend`) for any scope change:

```bash
prumo goal amend P00-G01 --file amendment.json
```

---

## 5. Connector Management (`prumo connector`)

```bash
# List all supported connectors
prumo connector list

# Inspect status, tools on the PATH, and capabilities
prumo connector status claude
prumo connector status opencode
prumo connector status antigravity

# Install a connector in the global environment (~/.prumo/connectors/)
prumo connector install opencode
```

---

## 6. Compile Adapters for AI Agents (`prumo compile`)

```bash
# Compile all configured adapters
prumo compile --all

# Compile for specific agents
prumo compile --target claude-code
prumo compile --target antigravity
prumo compile --target cursor
prumo compile --target opencode
```

The generated files (`CLAUDE.md`, `.gemini/`, `.cursorrules`) use `doccompile` delimiter markers and preserve any manual instructions from the developers.

---

## 7. Interactive Coding Agent (`prumo agent`)

```bash
# Start the interactive terminal interface (TUI)
prumo agent

# Run a specific goal autonomously
prumo run
```

---

## 8. Evidence and Intelligence Reports

```bash
# Record task execution evidence
prumo report add task-report.json

# Query the project's intelligence metrics and history
prumo report summary
prumo --json report summary
```

---

## 9. Unified JSON Contract for Code Agents

All commands support the global `--json` flag:

```bash
prumo --json version
prumo --json validate
prumo --json doctor
prumo --json status
```

Standardized envelope returned on standard output:

```json
{
  "protocol_version": "1",
  "ok": true,
  "data": {},
  "diagnostics": [],
  "warnings": []
}
```

CLI exit codes:

| Code | Meaning |
|---|---|
| `0` | Success / Operation completed |
| `1` | Schema, gate, or evidence validation failed |
| `2` | Usage error or invalid arguments |
| `3` | Configuration error |
| `4` | Capability or service unavailable |
| `5` | Internal error |
| `6` | Prumo project not found |

---

## 10. Updating Prumo (`prumo upgrade`)

```bash
# Check for new versions
prumo upgrade --check

# Update the global binary
prumo upgrade
```
