# Compiler Internals

Prumo compiles Goals and workforce into adapter-specific formats for different orchestrators and IDEs.

## Overview

The compiler transforms the canonical JSON/Markdown project representation into formats that each platform understands:

```
prumo.json + .ai/goals/ + .ai/agents/ + .ai/skills/
                              ↓
                        compile_target(target)
                              ↓
              .codex/ | .claude/ | .traycer/ | .prumo/runtime/
```

**No platform-specific data pollutes the canonical project.** The adapter is generated at compile time.

## Supported Targets

### 1. **generic** (Default, Provider-Agnostic)

**Purpose:** Standalone runtime entrypoint for any LLM orchestrator.

**Output:** `.prumo/runtime/compiled/generic/ENTRYPOINT.md`

Contains:
- Framework invariants
- Lean Progressive Context instructions
- Selected agents (by ID)
- Selected skills (by ID)
- Pointer to active Goal

**Use case:** Local CLI, custom automation, generic API clients.

```bash
prumo-agent compile --target generic
```

---

### 2. **codex** (GitHub Copilot)

**Purpose:** GitHub Copilot Chat integration with workspace-aware context.

**Output:**
- `.codex/AGENTS.md` — Copilot-specific agent instructions
- `.codex/agents/{agent_id}.md` — Per-agent markdown
- `.codex/skills/{skill_id}/` — Skill package directory

**Features:**
- Workspace context awareness
- GitHub Copilot Chat slash commands integration
- Real-time agent/skill discovery from `.codex/` tree

**Use case:** GitHub Copilot Chat in VS Code/GitHub.com.

```bash
prumo-agent compile --target codex
# Creates .codex/ with agent/skill tree
```

---

### 3. **claude-code** (Claude Code Tool)

**Purpose:** Claude Code (Claude API code execution) integration.

**Output:**
- `.claude/CLAUDE.md` — Claude-specific instructions
- `.claude/agents/{agent_id}.md` — Per-agent prompts
- `.claude/skills/{skill_id}/` — Skill package directory

**Features:**
- File-aware context
- Code execution sandbox preparation
- Tool use schema declarations

**Use case:** Claude Code (via Anthropic API or IDE).

```bash
prumo-agent compile --target claude-code
# Creates .claude/ with agent/skill tree
```

---

### 4. **chatgpt** (OpenAI ChatGPT)

**Purpose:** OpenAI ChatGPT / GPT-4 integration.

**Output:** `.prumo/runtime/compiled/chatgpt/ENTRYPOINT.md`

Contains:
- LPC/PCA instructions
- Agent roster
- Skill catalog
- Active Goal pointer

**Features:**
- Stateless prompt format
- Custom instructions compatible
- No persistent file tree

**Use case:** ChatGPT web/API, custom integrations.

```bash
prumo-agent compile --target chatgpt
```

---

### 5. **kimi** (Moonshot AI / Kimi)

**Purpose:** Kimi LLM (Chinese market) integration.

**Output:** `.prumo/runtime/compiled/kimi/ENTRYPOINT.md`

Contains:
- LPC instructions adapted for Kimi
- Agent roster
- Skill catalog
- Active Goal

**Features:**
- Long-context support
- Web search integration preparation
- Chinese language optimization

**Use case:** Moonshot Kimi API/Chat.

```bash
prumo-agent compile --target kimi
```

---

### 6. **traycer** (Anthropic Trace Agent Runtime)

**Purpose:** Anthropic's internal trace/audit system for agent execution.

**Output:** `.traycer/PROJECT_PRUMO.md`

Contains:
- Execution trace format
- Selected agents + skills
- LPC/PCA strategy
- Audit trail template

**Features:**
- Execution tracing
- Cost tracking
- Decision audit trail
- Token accounting

**Use case:** Internal Anthropic execution, multi-turn reasoning traces.

```bash
prumo-agent compile --target traycer
```

---

### 7. **claude** (Raw Claude prompt)

**Purpose:** Direct Claude API (non-Code tool) integration.

**Output:** `.prumo/runtime/compiled/claude/ENTRYPOINT.md`

Contains:
- Raw Claude system prompt format
- Agent/skill instructions
- Goal context
- LPC guidelines

**Features:**
- Minimal tooling assumptions
- Raw instruction text
- Vision + text support ready

**Use case:** Anthropic Claude API direct calls.

```bash
prumo-agent compile --target claude
```

---

## Compilation Process

### Step 1: Load Project

```python
root = Path(project_path)
prumo_json = load_data(root / "prumo.json")
agents_manifest = load_data(root / ".ai/agents/manifest.json")
skills_manifest = load_data(root / ".ai/skills/manifest.json")
```

### Step 2: Resolve Selected Agents & Skills

```python
selected_agents = agents_manifest.get("agents", [])  # List of agent IDs
selected_skills = skills_manifest.get("skills", [])  # List of skill IDs
```

### Step 3: Render Target-Specific Format

For **codex** / **claude-code**: Generate `.{target}/agents/{id}.md` + `.{target}/skills/{id}/` tree
For **generic** / **chatgpt** / **claude** / **kimi**: Generate `.prumo/runtime/compiled/{target}/ENTRYPOINT.md`
For **traycer**: Generate `.traycer/PROJECT_PRUMO.md`

### Step 4: Preserve Canonical

Original project remains **untouched**:
- ✅ `prumo.json` unchanged
- ✅ `.ai/` structure unchanged
- ✅ No YAML artifacts generated (legacy v0.3+ policy)

---

## Adapter Format Reference

Each adapter is stored in `src/prumo/resources/adapters/{target}.md`:

```
resources/
├── adapters/
│   ├── generic.md          (provider-agnostic runtime)
│   ├── codex.md            (GitHub Copilot Chat)
│   ├── claude-code.md      (Claude Code tool)
│   ├── claude.md           (Raw Claude API)
│   ├── chatgpt.md          (OpenAI ChatGPT)
│   ├── kimi.md             (Moonshot Kimi)
│   └── traycer.md          (Anthropic trace runtime)
├── workforce/
│   ├── agents/
│   │   ├── implementer/AGENT.md
│   │   ├── reviewer/AGENT.md
│   │   └── ...
│   └── skills/
│       ├── clean-code/
│       │   ├── SKILL.md
│       │   ├── rules.md
│       │   └── examples/
│       └── ...
```

---

## Example: Compiling for Codex

```bash
$ prumo compile --target codex --path my-project

# Creates:
# .codex/AGENTS.md
# .codex/agents/implementer.md
# .codex/agents/reviewer.md
# .codex/skills/clean-code/SKILL.md
# .codex/skills/clean-code/rules.md
# ... (all selected agents + skills)
```

---

## Example: Compiling for Claude Code

```bash
$ prumo compile --target claude-code --path my-project

# Creates:
# .claude/CLAUDE.md
# .claude/agents/implementer.md
# .claude/agents/reviewer.md
# .claude/skills/clean-code/SKILL.md
# .claude/skills/clean-code/rules.md
# ... (all selected agents + skills)
```

---

## Determinism & Performance

**Determinism:** Compilation must produce identical output byte-for-byte across runs. Sorting, hashing, and copying operations ensure this.

**Performance:** O(n) where n = number of selected agents + skills. Recompilation is fast and safe.

**Best practice:** Recompile frequently. Adapters are derived artifacts and safe to regenerate at any time.

---

## Troubleshooting

**Q: "Unsupported target: xyz"**
A: Use one of: `generic`, `codex`, `claude-code`, `claude`, `chatgpt`, `kimi`, `traycer`

**Q: No `.codex/` directory created**
A: Ensure `.ai/agents/manifest.json` exists and lists agent IDs:
```json
{ "agents": ["implementer", "reviewer"] }
```

**Q: Agent markdown is missing content**
A: If `resources/workforce/agents/{id}/AGENT.md` doesn't exist, a fallback from `catalog.json` is rendered. Add the package to `resources/workforce/agents/` for full content.

---

## See Also

- [Workforce Resolution](./resolver.md) — How agents + skills are selected
- [Lean Progressive Context](../LEAN_PROGRESSIVE_CONTEXT.md) — Context strategy for each adapter
- [Integration Guide](../integration/generic.md) — Using compiled output
