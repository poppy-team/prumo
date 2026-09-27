---
layout: home

hero:
  name: Prumo
  text: Engineering Harness & Git-Native Protocol
  tagline: Executable control plane for autonomous and collaborative software engineering with AI agents and human developers. 189 precision skills, 39 specialized agents, SHA-256 integrity locks, and Lean Progressive Context (LPC).
  image:
    src: /assets/logo.svg
    alt: Prumo Framework Logo
  actions:
    - theme: brand
      text: Get Started (5 min)
      link: /en/getting-started/
    - theme: alt
      text: Workforce Catalog
      link: /en/workforce/
    - theme: alt
      text: Harness & ACI
      link: /en/harness/
    - theme: alt
      text: CLI Reference
      link: /en/tools/

features:
  - icon: 🎯
    title: Cognitive Clarity & LPC Methodology
    details: Lean Progressive Context (LPC) — delivers strictly the minimal necessary context to LLMs. Progressive expansion, bounded outputs, and zero context pollution.
    link: /en/governance/
  - icon: 🛠️
    title: Executable Harness & Built-in ACI
    details: Integrated Tool Gateway with deterministic sandboxing, AST parsers, process execution, strict timeouts, and automated validation gates.
    link: /en/harness/
  - icon: 🛡️
    title: Strict Governance & Integrity Locks
    details: Goals and Plans protected by SHA-256 digests. Formal git-audited amendments preventing silent prompt mutations during agent runs.
    link: /en/getting-started/
  - icon: 🤖
    title: Canonical Workforce Registry
    details: 189 cataloged skills, 39 specialized agents (systems, architecture, design systems, UI/UX, SVG/motion, security), and 16 deterministic recipes.
    link: /en/workforce/
  - icon: 🔌
    title: Universal Provider-Neutral Connectors
    details: Surgical adapter compilers for Google Antigravity, Claude Code, OpenAI Codex, Cursor, Windsurf, OpenCode, Cline, and Roo Code.
    link: /en/tools/
  - icon: 📜
    title: Durable Memory & Living Book
    details: End-to-end traceability (Requirements ↔ ADRs ↔ Code ↔ Tests ↔ Evidence), episodic session memory, and living synchronization with the Living Book.
    link: /en/governance/
---

<div class="vp-doc">

## Evidence-Driven Autonomous Engineering

**Prumo** brings determinism, verifiable evidence, and cryptographic integrity to AI-assisted software development. Rather than relying on transient chat sessions and fragile megaprompts, Prumo establishes a **Git-native control plane** where engineering intent compiles cleanly into verifiable code, documentation, and audited evidence.

```bash
# Initialize a project with a canonical profile
prumo init ./my-project --profile ./profiles/web-service.json

# Create and lock a Goal with SHA-256 integrity
prumo goal new P01-G01 "Core Authentication" --phase P01
prumo goal state P01-G01 LOCKED

# Run directives in a deterministic sandboxed harness
prumo run --goal P01-G01

# Audit documentation authority and drift
prumo docs audit
```

---

## All Developer Tools in a Single Go Binary

Prumo v0.6 is a self-contained single-binary distribution with zero external runtime dependencies.

<div class="tool-grid">

<div class="tool-card">
  <div>
    <h3>▶️ Run Engine</h3>
    <p>Execute directives, tasks, and plans within sandboxed execution environments.</p>
  </div>
  <div class="tool-cmd">prumo run --path .</div>
</div>

<div class="tool-card">
  <div>
    <h3>🎯 Goals & DAGs</h3>
    <p>Manage formal Goal lifecycles with SHA-256 locks and non-bypassable amendments.</p>
  </div>
  <div class="tool-cmd">prumo goal list</div>
</div>

<div class="tool-card">
  <div>
    <h3>🔌 Compile Adapters</h3>
    <p>Generate surgical rule files for Claude Code, Antigravity, Cursor, and Windsurf.</p>
  </div>
  <div class="tool-cmd">prumo compile --target all</div>
</div>

<div class="tool-card">
  <div>
    <h3>🩺 Repository Doctor</h3>
    <p>Deep diagnostic checks verifying schemas, locks, DAGs, and harness health.</p>
  </div>
  <div class="tool-cmd">prumo doctor</div>
</div>

</div>

</div>
