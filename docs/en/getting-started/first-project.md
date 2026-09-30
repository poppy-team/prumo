# Your First Project with Prumo (5 minutes)

Initializing a project in Prumo is instant and does not require writing any configuration files by hand.

---

## 1. Zero-Configuration Initialization (Greenfield)

To create a new project governed by Prumo, just create the directory and run `prumo init`:

```bash
mkdir my-app && cd my-app
git init
prumo init
```

Prumo will detect the directory and create the canonical **Protocol v3** structure:

```text
my-app/
├── prumo.json                            # Canonical project configuration (v3)
├── PROJECT_STATE.md                      # Current state, phase, and metadata
├── ENTRYPOINT.md                         # Onboarding guide for agents and humans
├── docs/
│   └── PRUMO.md                          # Documentation and intent router
├── .ai/
│   ├── agents/manifest.json              # Resolved agent workforce
│   ├── skills/manifest.json              # Required skills
│   ├── recipes/manifest.json             # Engineering recipes
│   └── orchestration/                    # Fallback policies and scorecards
└── .prumo/                               # Local runtime, intelligence history, and cache
```

---

## 2. Initialization with Presets and Flags

If you want to customize the project's stack or focus right away:

```bash
# Preset for Web & Frontend applications
prumo init --preset web --name customer-portal

# Preset for high-performance microservices and APIs
prumo init --preset service --stack go

# Preset for command-line tools (CLI)
prumo init --preset cli --name report-generator

# Inspect the generated profile before applying
prumo init --print-profile
```

### Available Presets:
- **`standard`** (default): Full software engineering with balanced quality and coverage.
- **`cli`**: Focus on terminal experience, TUI interfaces, and command-line flags.
- **`web`**: Focus on responsive design, visual components, and WCAG accessibility.
- **`service`**: Resilient APIs, microservices, and hardened security.
- **`library`**: Reusable packages and code with zero external dependencies.
- **`minimal`**: Lightweight configuration with minimal token consumption for rapid prototyping.

---

## 3. Verify Conformance

After initializing, validate your project with the audit command:

```bash
prumo validate
```

The command will confirm that all JSON Schema definitions, canonical files, and Protocol v3 contracts are 100% conformant.

---

## 4. Compile Adapters for Your AI Agents

So that Claude Code, Google Antigravity, OpenCode, or Cursor understand your project's guidelines:

```bash
# Compile adapters for all supported environments
prumo compile --all

# Or compile for a specific environment
prumo compile --target claude-code
prumo compile --target antigravity
```

---

## 5. Start the Interactive Coding Agent (TUI)

Prumo includes a built-in terminal agent with a rich interface:

```bash
prumo agent
```

You can chat with the agent, request implementations, and follow the architectural decisions with full traceability.
