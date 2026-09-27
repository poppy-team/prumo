#!/usr/bin/env python3
"""
Scaffolds advanced workforce skills and agents for low-level systems,
compilers, virtual machines, high-performance sciences, and systems security.
"""

import os
import json

BASE_SKILLS_DIR = "src/prumo/resources/workforce/skills"
BASE_AGENTS_DIR = "src/prumo/resources/workforce/agents"
CATALOG_SKILLS_PATH = "src/prumo/resources/catalog/skills.json"
CATALOG_AGENTS_PATH = "src/prumo/resources/catalog/agents.json"
CATALOG_BUNDLES_PATH = "src/prumo/resources/catalog/bundles.json"
CATALOG_RECIPES_PATH = "src/prumo/resources/catalog/recipes.json"

NEW_SKILLS = [
    {
        "id": "compiler-frontend-engineering",
        "name": "Compiler Frontend Engineering",
        "purpose": "Design and implement lexing, parsing (Pratt, LR, PEG), AST/CST construction, and robust error recovery.",
        "risk_level": "medium",
        "modes": ["implementation", "testing", "review"],
        "inputs": ["Formal grammar specification", "Lexer token definition", "Source code files"],
        "outputs": ["Concrete and Abstract Syntax Trees", "Diagnostic error reports with source spans"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["test", "conformance"],
        "select": {"project_types": ["compiler", "language", "game-engine"], "features_any": ["compiler", "parser", "dsl"]}
    },
    {
        "id": "type-system-theory",
        "name": "Type System Theory & Formal Semantics",
        "purpose": "Specify and verify type systems, Hindley-Milner inference, subtyping, and ownership/affine type rules.",
        "risk_level": "high",
        "modes": ["design", "implementation", "review"],
        "inputs": ["Type rules specification", "AST with unresolved types"],
        "outputs": ["Typed AST", "Soundness verification proofs", "Type error diagnostics"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["test", "review"],
        "select": {"project_types": ["compiler", "language"], "features_any": ["type-checker", "compiler"]}
    },
    {
        "id": "compiler-ir-optimization",
        "name": "Compiler IR & SSA Optimization",
        "purpose": "Transform AST into SSA form, compute dominance frontiers, and perform dead code elimination, GVN, and vectorization.",
        "risk_level": "high",
        "modes": ["implementation", "review", "testing"],
        "inputs": ["Typed AST or high-level IR", "Target optimization budgets"],
        "outputs": ["Optimized SSA Intermediate Representation", "Control Flow Graph (CFG) analysis"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["test", "benchmark"],
        "select": {"project_types": ["compiler", "language"], "features_any": ["optimization", "compiler-backend"]}
    },
    {
        "id": "bytecode-vm-architecture",
        "name": "Bytecode & Virtual Machine Architecture",
        "purpose": "Design stack and register-based virtual machines, direct threaded dispatch loops, and bytecode verifiers.",
        "risk_level": "high",
        "modes": ["implementation", "testing", "review"],
        "inputs": ["Bytecode instruction set architecture (ISA)", "Compiled bytecode binary"],
        "outputs": ["Virtual machine runtime", "Bytecode execution trace", "Execution verification evidence"],
        "capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test", "benchmark"],
        "select": {"project_types": ["compiler", "language", "game-engine"], "features_any": ["vm", "bytecode", "interpreter"]}
    },
    {
        "id": "runtime-memory-gc",
        "name": "Runtime Memory Management & Garbage Collection",
        "purpose": "Implement custom allocators (arenas, pools, TLSF) and garbage collectors (generational, incremental, concurrent).",
        "risk_level": "high",
        "modes": ["implementation", "testing", "review"],
        "inputs": ["Memory budget constraints", "Runtime allocation profile"],
        "outputs": ["Memory manager implementation", "Allocation profiling reports", "Leak and fragmentation metrics"],
        "capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test", "benchmark"],
        "select": {"project_types": ["compiler", "language", "game-engine"], "features_any": ["memory-management", "runtime"]}
    },
    {
        "id": "computational-physics",
        "name": "Computational Physics & Symplectic Dynamics",
        "purpose": "Model rigid and soft body physics using symplectic integrators (Verlet, RK4) and GJK/EPA continuous collision.",
        "risk_level": "medium",
        "modes": ["implementation", "testing", "review"],
        "inputs": ["Physical body definitions", "Simulation boundary parameters", "Time step configuration"],
        "outputs": ["Physics simulation state", "Collision manifold records", "Energy conservation metrics"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["test", "benchmark"],
        "select": {"project_types": ["game-engine", "desktop"], "features_any": ["physics", "simulation"]}
    },
    {
        "id": "applied-mathematics-dsp",
        "name": "Applied Mathematics & Digital Signal Processing",
        "purpose": "Execute numerical computations, quaternion geometry, and Fast Fourier Transforms for graphics, audio, and CAD.",
        "risk_level": "low",
        "modes": ["implementation", "testing", "review"],
        "inputs": ["Mathematical equations", "Audio/visual time-domain buffers", "Transform specifications"],
        "outputs": ["DSP pipeline implementation", "Frequency spectra", "Numerical stability proofs"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["test"],
        "select": {"project_types": ["game-engine", "desktop"], "features_any": ["dsp", "audio", "graphics", "math"]}
    },
    {
        "id": "computational-chemistry-materials",
        "name": "Computational Chemistry & Physically Based Materials",
        "purpose": "Simulate material optical properties (complex refractive index, Fresnel, microfacets) and molecular potential interactions.",
        "risk_level": "low",
        "modes": ["implementation", "research", "review"],
        "inputs": ["Material spectral specifications", "Refractive index datasets", "Molecular parameters"],
        "outputs": ["PBR material shader parameters", "Spectral absorption profiles"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["review", "test"],
        "select": {"project_types": ["game-engine", "desktop"], "features_any": ["rendering", "shaders", "materials"]}
    },
    {
        "id": "realtime-audio-dsp",
        "name": "Realtime Audio DSP & Zero-Allocation Pipelines",
        "purpose": "Build ultra-low latency, lock-free audio processing graphs with strict zero-allocation in the realtime audio thread.",
        "risk_level": "medium",
        "modes": ["implementation", "testing", "review"],
        "inputs": ["Audio buffer constraints", "Sample rate specifications", "DSP filter graphs"],
        "outputs": ["Lock-free audio engine", "Underrun-free performance benchmarks"],
        "capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test", "benchmark"],
        "select": {"project_types": ["game-engine", "desktop"], "features_any": ["audio", "dsp", "realtime"]}
    },
    {
        "id": "memory-safety-sanitizers",
        "name": "Native Memory Safety & Sanitizer Verification",
        "purpose": "Verify C/C++/Rust native code against UAF, buffer overflows, data races, and leaks using ASan, MSan, TSan, and UBSan.",
        "risk_level": "high",
        "modes": ["testing", "review", "audit"],
        "inputs": ["Native source code", "Compilation flags", "Test binaries"],
        "outputs": ["Sanitizer execution report", "Memory safety violation logs"],
        "capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test", "review"],
        "select": {"project_types": ["compiler", "game-engine", "desktop", "systems"], "risk_any": ["security", "performance-critical"]}
    },
    {
        "id": "fuzz-grammar-testing",
        "name": "Grammar-Based Fuzzing & Property Verification",
        "purpose": "Execute coverage-guided and grammar-based fuzzing on parsers, decoders, and compilers using libFuzzer and AFL++.",
        "risk_level": "high",
        "modes": ["testing", "audit"],
        "inputs": ["Target API / parser entry point", "Input grammar dictionary", "Seed corpus"],
        "outputs": ["Crash reproduction testcases", "Corpus coverage report"],
        "capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test"],
        "select": {"project_types": ["compiler", "game-engine", "systems"], "risk_any": ["security"]}
    },
    {
        "id": "binary-security-mitigations",
        "name": "Binary Hardening & Exploit Mitigations",
        "purpose": "Enforce Control Flow Integrity (CFI), Stack Canaries, SafeStack, RELRO, and ASLR compliance on compiled artifacts.",
        "risk_level": "high",
        "modes": ["audit", "release", "review"],
        "inputs": ["Compiled binary executables", "Linker configurations"],
        "outputs": ["Binary hardening scorecard", "Mitigation verification report"],
        "capabilities": ["filesystem.read", "process.spawn"],
        "required_evidence": ["review", "test"],
        "select": {"project_types": ["compiler", "desktop", "systems"], "risk_any": ["security"]}
    },
    {
        "id": "linux-kernel-isolation",
        "name": "Linux Kernel Isolation & Sandboxing",
        "purpose": "Enforce process sandboxing and isolation via seccomp-BPF, Landlock LSM, cgroups v2, and Linux namespaces.",
        "risk_level": "high",
        "modes": ["implementation", "audit", "testing"],
        "inputs": ["Sandbox security profile", "Permitted syscalls list", "Filesystem access mask"],
        "outputs": ["Seccomp/Landlock policy rules", "Isolation verification test suite"],
        "capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test", "review"],
        "select": {"features_any": ["security", "sandboxing", "isolation"], "risk_any": ["security"]}
    },
    {
        "id": "security-threat-model",
        "name": "Structured Threat Modeling & STRIDE",
        "purpose": "Map system assets, trust boundaries, attacker capabilities, STRIDE threat matrices, and mitigation requirements.",
        "risk_level": "high",
        "modes": ["design", "review", "audit"],
        "inputs": ["Architecture diagrams", "Data classification", "Trust boundary specifications"],
        "outputs": ["Threat model document", "Abuse cases catalog", "Residual risk ledger"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["review"],
        "select": {"features_any": ["security", "auth", "billing"], "risk_any": ["security", "privacy"]}
    },
    {
        "id": "security-authz-matrix",
        "name": "Authorization Matrix & Multi-Tenant Enforcement",
        "purpose": "Define and automatically verify fine-grained role-resource-action authorization matrices and cross-tenant boundaries.",
        "risk_level": "high",
        "modes": ["design", "testing", "audit"],
        "inputs": ["API route inventory", "Tenant data schema", "Permission definitions"],
        "outputs": ["Authorization test suite", "Negative access control audit report"],
        "capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test", "review"],
        "select": {"features_any": ["security", "auth", "multi-tenant"], "risk_any": ["security"]}
    },
    {
        "id": "security-saas-isolation",
        "name": "SaaS Multi-Tenant Isolation Verification",
        "purpose": "Execute adversarial cross-user and cross-tenant penetration tests across database RLS, caches, queues, and storage.",
        "risk_level": "critical",
        "modes": ["testing", "audit"],
        "inputs": ["Multi-tenant test fixtures", "Target API endpoints"],
        "outputs": ["Cross-tenant isolation audit report", "Data leak verification evidence"],
        "capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test"],
        "select": {"features_any": ["multi-tenant", "database"], "risk_any": ["security", "privacy"]}
    },
    {
        "id": "security-agent-mcp",
        "name": "Agentic Security & MCP Tool Sandboxing",
        "purpose": "Audit coding agents, MCP tool manifests, prompt injection vectors, permission drift, and tool shadowing.",
        "risk_level": "critical",
        "modes": ["audit", "review", "testing"],
        "inputs": ["MCP server configurations", "Agent system instructions", "Skill manifests"],
        "outputs": ["Agent security scorecard", "MCP sandbox policy manifest"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["review", "test"],
        "select": {"features_any": ["mcp", "ai-assisted-development", "orchestration"], "risk_any": ["security"]}
    }
]

NEW_AGENTS = [
    {
        "id": "systems-architect",
        "name": "Systems Architect",
        "version": 2,
        "purpose": "Architect low-level systems, compiler backends, VMs, frame/audio budgets, memory models, and cache-conscious layouts.",
        "risk_level": "high",
        "select": {
            "project_types": ["compiler", "language", "game-engine", "desktop", "systems"]
        },
        "inputs": [
            "Hardware performance budgets (latency, throughput, memory)",
            "System boundary requirements",
            "Target ABI/ISA specifications"
        ],
        "outputs": [
            "Systems Architecture Decision Record (ADR)",
            "Memory & Allocation Topology specification",
            "Concurrency and Cache Invariants contract"
        ],
        "required_skills": [
            "architecture-quality",
            "clean-code",
            "memory-management",
            "concurrency-quality"
        ],
        "optional_skills": [
            "performance-native",
            "bytecode-vm-architecture",
            "compiler-ir-optimization",
            "binary-security-mitigations"
        ],
        "allowed_capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["review"],
        "may_delegate": False,
        "max_delegation_depth": 0,
        "handoff_to": ["compiler-engineer", "engine-engineer", "implementer"],
        "review_requirement": "cross-provider",
        "permissions": {"write_code": True, "modify_docs": True, "execute_tests": True},
        "stop_conditions": [
            "Systems architecture approved with formal memory, latency, and concurrency invariants"
        ],
        "required_capabilities": ["filesystem.read", "filesystem.write", "process.spawn"]
    },
    {
        "id": "isolation-auditor",
        "name": "Isolation Auditor",
        "version": 2,
        "purpose": "Audit and verify multi-tenant isolation, process containment, capability boundaries, sandbox policies, and abuse resistance.",
        "risk_level": "high",
        "select": {
            "features_any": ["security", "multi-tenant", "sandboxing", "plugins", "mcp"],
            "risk_any": ["security", "privacy"]
        },
        "inputs": [
            "Tenant isolation policy",
            "Process sandbox manifests",
            "Tool execution policy"
        ],
        "outputs": [
            "Isolation penetration test report",
            "Sandbox escape audit findings",
            "Adversarial abuse scorecard"
        ],
        "required_skills": [
            "security-review",
            "untrusted-project-security",
            "secrets-security",
            "security-saas-isolation"
        ],
        "optional_skills": [
            "linux-kernel-isolation",
            "security-agent-mcp",
            "process-execution-security"
        ],
        "allowed_capabilities": ["filesystem.read", "filesystem.write", "process.spawn"],
        "required_evidence": ["test", "review"],
        "may_delegate": False,
        "max_delegation_depth": 0,
        "handoff_to": ["security-reviewer", "reviewer"],
        "review_requirement": "mandatory",
        "permissions": {"write_code": False, "modify_docs": True, "execute_tests": True},
        "stop_conditions": [
            "Cross-tenant and sandbox isolation verified with zero high-severity leaks"
        ],
        "required_capabilities": ["filesystem.read", "process.spawn"]
    }
]

def scaffold_skill(skill):
    sid = skill["id"]
    sdir = os.path.join(BASE_SKILLS_DIR, sid)
    for sub in ["checks", "examples", "references", "scripts", "templates"]:
        os.makedirs(os.path.join(sdir, sub), exist_ok=True)
    
    # 1. manifest.json
    manifest = {
        "id": sid,
        "name": skill["name"],
        "version": 2,
        "schema_version": 3,
        "purpose": skill["purpose"],
        "risk_level": skill["risk_level"],
        "modes": skill["modes"],
        "inputs": skill["inputs"],
        "outputs": skill["outputs"],
        "requires": [],
        "conflicts": [],
        "capabilities": skill["capabilities"],
        "references": [f"references/{sid}-guide.md"],
        "templates": [f"templates/{sid}-spec.md"],
        "checks": [f"checks/{sid}-checklist.md"],
        "scripts": ["scripts/verify.sh"],
        "examples": [f"examples/{sid}-example.json"],
        "required_evidence": skill["required_evidence"],
        "stop_conditions": [f"All {skill['name']} invariants verified with evidence"],
        "provenance": {
            "origin": "framework",
            "license": "MIT",
            "source": "prumo",
            "version": "0.4.0",
            "checksum": "",
            "modified": "2026-09-23"
        },
        "select": skill.get("select", {}),
        "requires_any": [],
        "tools": []
    }
    with open(os.path.join(sdir, "manifest.json"), "w", encoding="utf-8") as f:
        json.dump(manifest, f, indent=2)
        f.write("\n")

    # 2. SKILL.md
    skill_md = f"""---
name: {sid}
description: {skill['purpose']}
---
# {skill['name']}

## Purpose & Rigor
{skill['purpose']}

## Operational Invariants
1. **Academic Rigor & First Principles:** Prioritize formal correctness, proof of termination, memory safety, and algorithmic complexity over convenience.
2. **Deterministic Evidence:** Never declare success without verifiable test, benchmark, or formal audit evidence.
3. **Fail-Closed Security:** In the presence of ambiguity or missing specifications, deny access and reject unsafe operations.
"""
    with open(os.path.join(sdir, "SKILL.md"), "w", encoding="utf-8") as f:
        f.write(skill_md)

    # 3. checks, references, templates, examples
    with open(os.path.join(sdir, f"references/{sid}-guide.md"), "w", encoding="utf-8") as f:
        f.write(f"# Guide: {skill['name']}\n\nTechnical guidelines and theoretical foundations for `{sid}`.\n")
    
    with open(os.path.join(sdir, f"templates/{sid}-spec.md"), "w", encoding="utf-8") as f:
        f.write(f"# Specification Template: {skill['name']}\n\nFormal specification template for `{sid}`.\n")

    with open(os.path.join(sdir, f"checks/{sid}-checklist.md"), "w", encoding="utf-8") as f:
        f.write(f"# Checklist: {skill['name']}\n\n- [ ] Invariants verified\n- [ ] Evidence recorded\n")

    with open(os.path.join(sdir, f"examples/{sid}-example.json"), "w", encoding="utf-8") as f:
        json.dump({"skill": sid, "status": "verified"}, f, indent=2)
        f.write("\n")

    # 4. scripts/verify.sh
    verify_sh = f"""#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="{sdir}"
echo "[Prumo Skill: {sid}] Asset verification"
for file in "SKILL.md" "manifest.json" "references/{sid}-guide.md" "templates/{sid}-spec.md" "checks/{sid}-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] {sid} verification passed"
exit 0
"""
    verify_path = os.path.join(sdir, "scripts/verify.sh")
    with open(verify_path, "w", encoding="utf-8") as f:
        f.write(verify_sh)
    os.chmod(verify_path, 0o755)

def scaffold_agent(agent):
    aid = agent["id"]
    adir = os.path.join(BASE_AGENTS_DIR, aid)
    os.makedirs(adir, exist_ok=True)
    
    # 1. manifest.json
    with open(os.path.join(adir, "manifest.json"), "w", encoding="utf-8") as f:
        json.dump(agent, f, indent=2)
        f.write("\n")

    # 2. AGENT.md
    agent_md = f"""# {agent['name']}

## Purpose
{agent['purpose']}

## Inputs
{chr(10).join(f"- **REQUIRED — {inp}**" for inp in agent['inputs'])}

## Outputs
{chr(10).join(f"- **{out}**" for out in agent['outputs'])}

## Required Skills
{chr(10).join(f"- `{sk}`" for sk in agent['required_skills'])}

## Capabilities & Permissions
- **Risk Level:** `{agent['risk_level']}`
- **Allowed Capabilities:** {", ".join(f"`{c}`" for c in agent['allowed_capabilities'])}
- **Review Requirement:** `{agent['review_requirement']}`

## Invariants & What NOT To Do (Must Not)
- Never bypass formal architectural, security, or physical invariants to make code compile quickly.
- Never declare a gate passed without reproducible evidence.
- Prioritize technical, mathematical, and algorithmic principles over informal user preferences.

## Stop Conditions
{chr(10).join(f"- {sc}" for sc in agent['stop_conditions'])}
"""
    with open(os.path.join(adir, "AGENT.md"), "w", encoding="utf-8") as f:
        f.write(agent_md)

def update_catalogs():
    # Update skills.json
    with open(CATALOG_SKILLS_PATH, "r", encoding="utf-8") as f:
        cdata = json.load(f)
    existing_skills = {s["id"]: s for s in cdata["skills"]}
    for sk in NEW_SKILLS:
        sid = sk["id"]
        if sid not in existing_skills:
            skill_entry = {
                "id": sid,
                "name": sk["name"],
                "version": 2,
                "schema_version": 3,
                "purpose": sk["purpose"],
                "risk_level": sk["risk_level"],
                "modes": sk["modes"],
                "inputs": sk["inputs"],
                "outputs": sk["outputs"],
                "requires": [],
                "conflicts": [],
                "capabilities": sk["capabilities"],
                "references": [f"references/{sid}-guide.md"],
                "templates": [f"templates/{sid}-spec.md"],
                "checks": [f"checks/{sid}-checklist.md"],
                "scripts": ["scripts/verify.sh"],
                "examples": [f"examples/{sid}-example.json"],
                "required_evidence": sk["required_evidence"],
                "stop_conditions": [f"All {sk['name']} invariants verified with evidence"],
                "provenance": {
                    "origin": "framework",
                    "license": "MIT",
                    "source": "prumo",
                    "version": "0.4.0",
                    "checksum": "",
                    "modified": "2026-09-23"
                },
                "select": sk.get("select", {}),
                "requires_any": [],
                "tools": []
            }
            cdata["skills"].append(skill_entry)
    cdata["skills"].sort(key=lambda s: s["id"])
    with open(CATALOG_SKILLS_PATH, "w", encoding="utf-8") as f:
        json.dump(cdata, f, indent=2)
        f.write("\n")

    # Update agents.json
    with open(CATALOG_AGENTS_PATH, "r", encoding="utf-8") as f:
        adata = json.load(f)
    existing_agents = {a["id"]: a for a in adata["agents"]}
    for ag in NEW_AGENTS:
        aid = ag["id"]
        if aid not in existing_agents:
            adata["agents"].append(ag)
    adata["agents"].sort(key=lambda a: a["id"])
    with open(CATALOG_AGENTS_PATH, "w", encoding="utf-8") as f:
        json.dump(adata, f, indent=2)
        f.write("\n")

    # Update bundles.json
    with open(CATALOG_BUNDLES_PATH, "r", encoding="utf-8") as f:
        bdata = json.load(f)
    existing_bundles = {b["id"]: b for b in bdata["bundles"]}
    
    new_bundles = [
        {
            "id": "compiler-toolchain",
            "project_type": "compiler",
            "skills": [
                "compiler-development",
                "compiler-frontend-engineering",
                "type-system-theory",
                "compiler-ir-optimization",
                "bytecode-vm-architecture",
                "runtime-memory-gc",
                "fuzz-grammar-testing",
                "clean-code"
            ],
            "agents": [
                "compiler-engineer",
                "systems-architect",
                "tester",
                "reviewer"
            ],
            "recipes": [
                "compiler-change",
                "feature-standard"
            ]
        },
        {
            "id": "high-performance-desktop",
            "project_type": "desktop",
            "skills": [
                "performance-native",
                "computational-physics",
                "applied-mathematics-dsp",
                "computational-chemistry-materials",
                "realtime-audio-dsp",
                "memory-management",
                "editor-tooling"
            ],
            "agents": [
                "systems-architect",
                "editor-engineer",
                "tester",
                "reviewer"
            ],
            "recipes": [
                "ui-feature",
                "feature-standard"
            ]
        },
        {
            "id": "systems-security",
            "project_type": "systems",
            "skills": [
                "memory-safety-sanitizers",
                "binary-security-mitigations",
                "linux-kernel-isolation",
                "security-threat-model",
                "security-authz-matrix",
                "security-saas-isolation",
                "security-agent-mcp",
                "secure-coding"
            ],
            "agents": [
                "security-architect",
                "security-reviewer",
                "isolation-auditor",
                "release-verifier"
            ],
            "recipes": [
                "security-review",
                "release"
            ]
        }
    ]
    for nb in new_bundles:
        if nb["id"] not in existing_bundles:
            bdata["bundles"].append(nb)
    with open(CATALOG_BUNDLES_PATH, "w", encoding="utf-8") as f:
        json.dump(bdata, f, indent=2)
        f.write("\n")

def main():
    print("Scaffolding skills...")
    for sk in NEW_SKILLS:
        scaffold_skill(sk)
        print(f"  + skill: {sk['id']}")

    print("Scaffolding agents...")
    for ag in NEW_AGENTS:
        scaffold_agent(ag)
        print(f"  + agent: {ag['id']}")

    print("Updating catalogs (skills.json, agents.json, bundles.json)...")
    update_catalogs()
    print("Done!")

if __name__ == "__main__":
    main()
