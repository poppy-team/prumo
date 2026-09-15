# Writing Skills

Prumo introduces Skill Packages v2, a directory-based structure that encapsulates everything an AI agent needs to execute a specific task reliably.

## Skill Package v2 Directory Structure

A skill package is a directory containing a `manifest.json` file, a `SKILL.md` instruction file, and optional supporting directories:

```text
skill-name/
├── manifest.json       # Formal capabilities and requirements
├── SKILL.md            # Natural language instructions for the LLM
├── checks/             # Validation checklists (e.g., pre-flight, post-flight)
├── templates/          # Code/text templates used by the skill
├── references/         # Example implementations or documentation
├── scripts/            # Helper scripts executed by the agent
└── examples/           # Example usage scenarios
```

## Manifest Reference (`manifest.json`)

The manifest defines the formal contract for the skill.

```json
{
  "id": "python-refactor",
  "name": "Python Refactoring Expert",
  "version": "1.0.0",
  "schema_version": "2.0",
  "purpose": "Refactors Python code for readability and performance.",
  "risk_level": "medium",
  "modes": ["autonomous", "interactive"],
  "inputs": [
    {"name": "file_path", "description": "Path to the Python file", "type": "path", "required": true}
  ],
  "outputs": [
    {"name": "diff", "description": "Proposed changes", "type": "string"}
  ],
  "requires": ["python3", "pytest"],
  "conflicts": ["legacy-python-refactor"],
  "capabilities": ["fs.read", "fs.write", "shell.exec"],
  "select": {
    "stack": ["python"],
    "features": ["refactoring"]
  },
  "required_evidence": ["tests_pass", "lint_clean"],
  "stop_conditions": ["Max iterations reached", "Tests fail after 3 attempts"],
  "provenance": {
    "author": "Prumo Team",
    "signature": "SHA256:xyz..."
  }
}
```

## SKILL.md Structure

The `SKILL.md` file must contain the following 10 sections:

1. **Title and Description:** Briefly explain what the skill does.
2. **Purpose:** The exact goal of the skill.
3. **Prerequisites:** What must be true before using this skill.
4. **Inputs:** Expected input variables.
5. **Outputs:** Expected outputs.
6. **Execution Steps:** Step-by-step instructions for the LLM.
7. **Verification:** How to verify the skill worked.
8. **Fallbacks:** What to do if execution fails.
9. **Constraints:** Rules the LLM must not break.
10. **Examples:** Concrete examples of good execution.

## Writing Supporting Files

### Checks (`checks/`)
Create markdown checklists (e.g., `pre-flight.md`) with checkboxes (`- [ ]`) that the agent must complete before or after execution.

### Templates (`templates/`)
Store boilerplate code or text formats the agent should use. Reference them in `SKILL.md`.

## Provenance and Supply Chain
Ensure every skill has an author and a mechanism to verify integrity (e.g., cryptographic signatures) in the `manifest.json`.

## Testing a New Skill
Run `prumo test skill <skill_id>` to validate the schema and run any associated unit tests or conformance checks in FakeRuntime.

## Resolver Selection
The resolver uses the `select` block in the manifest to match skills to the current project context (stack, features, etc.).

## Examples
* **Simple:** A skill that simply formats JSON.
* **Complex:** A skill that sets up a full PostgreSQL database, runs migrations, and verifies connectivity.
