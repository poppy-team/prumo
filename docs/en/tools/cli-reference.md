# Complete CLI Reference (`prumo`)

The `prumo` executable accepts commands and structured flags. For automation in CI pipelines or integration with agents, the `--json` flag guarantees standardized envelope output.

## Main Commands Table

| Command | Description | Usage Example |
|---|---|---|
| `prumo version` | Displays the CLI version, commit SHA, and build time. | `prumo version --json` |
| `prumo init <path>` | Initializes a Prumo repository with a canonical profile. | `prumo init ./my-project --profile ./profile.json` |
| `prumo goal new <id> <title>` | Creates a new Goal in the project. | `prumo goal new P01-G01 "Foundation"` |
| `prumo goal state <id> <state>` | Changes a Goal's state (`PLANNED`, `LOCKED`, etc.). | `prumo goal state P01-G01 LOCKED` |
| `prumo goal amend <id>` | Opens a formal amendment to change a locked Goal. | `prumo goal amend P01-G01 --reason "New scope"` |
| `prumo goal list` | Lists all Goals in the repository with status and phase. | `prumo goal list` |
| `prumo plan new` | Creates a new work Plan with a task DAG. | `prumo plan new --goal P01-G01` |
| `prumo run` | Runs tasks or directives through the Prumo Harness. | `prumo run --path .` |
| `prumo compile` | Compiles and syncs adapters for harnesses. | `prumo compile --target claude-code` |
| `prumo doctor` | Runs a battery of project integrity diagnostics. | `prumo doctor` |
| `prumo validate` | Validates all canonical files against JSON Schemas. | `prumo validate` |
| `prumo adopt` | Runs the adoption engine on legacy repositories. | `prumo adopt --path ./legacy --scan` |
| `prumo agent` | Launches the interactive prumo-agent client (or `pa`). | `prumo agent` |
| `prumo docs audit` | Audits documentation conformance and integrity. | `prumo docs audit` |
| `prumo docs authority` | Validates the authority hierarchy and routing links. | `prumo docs authority` |

## JSON Output Envelope (`--json`)

Every command invocation with the `--json` flag produces deterministic JSON on standard output (`stdout`), while warnings and debug logs are emitted on `stderr`:

```json
{
  "protocol_version": "1",
  "ok": true,
  "data": {
    "goal_id": "P01-G01",
    "status": "LOCKED",
    "lock_digest": "sha256:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"
  },
  "diagnostics": [],
  "warnings": []
}
```
