# Adopting Existing Projects (Brownfield)

Do you have an existing repository in Go, Node/TypeScript, Python, Rust, or another language and want to bring it into Prumo without rewriting code or losing history?

The **Adoption Engine (`prumo adopt`)** was designed for exactly that: it inspects technical facts, classifies the architecture, and generates a non-destructive canonical configuration.

---

## 1. The 2-Step Adoption Flow

### Step 1: Inspect and Audit

Run the `adopt` command at the root of your repository:

```bash
cd my-legacy-project
prumo adopt
```

Prumo will perform:
1. **Artifact scan**: detects manifests (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, Dockerfiles, CI/CD).
2. **Technical Fact Extraction**: languages, frameworks in use, databases, test suites.
3. **Architectural Classification**: determines the application types and suggests a capability profile.
4. **Migration Proposal**: shows a preview of the generated `prumo.json` **without changing any file on disk**.

### Step 2: Apply the Adoption

Once you are satisfied with the audit report, apply the adoption:

```bash
prumo adopt --apply
```

This command:
- Creates the `prumo.json` compatible with **Protocol v3**.
- Initializes the matching workforce under `.ai/`.
- Generates the intent-routing documentation in `docs/PRUMO.md`, `PROJECT_STATE.md`, and `ENTRYPOINT.md`.
- **Preserves 100% of your existing source code and documentation files**.

---

## 2. `prumo adopt` Subcommands

For code agents and CI/CD pipelines that need granular inspection:

| Command | Description |
|---|---|
| `prumo adopt scan [path]` | Indexes files, identifies known artifacts, and quantifies the repository. |
| `prumo adopt facts [path]` | Extracts and displays the ledger of observed technical facts with confidence levels. |
| `prumo adopt classify [path]` | Displays the pure architectural classification (languages, frameworks, toolchains). |
| `prumo adopt scaffold [path]` | Dry-run simulation of the proposed mutations with a diff preview. |
| `prumo adopt apply [path]` | Applies the proposals and configures the repository on Protocol v3. |

### JSON example for Code Agents:

```bash
prumo --json adopt classify .
```

---

## 3. Verifying the Adopted Project

After running `prumo adopt --apply`:

```bash
# Validate conformance
prumo validate

# Diagnose state and dependencies
prumo doctor

# Compile context adapters for your AI agents
prumo compile --all
```

Your existing repository is now fully protected against context drift and ready to work with autonomous engineering agents.
