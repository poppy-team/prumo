import os
import json

skills_dir = 'src/prumo/resources/workforce/skills'
catalog_skills_file = 'src/prumo/resources/catalog/skills.json'

p0_skills = [
    {
        "id": "grounded-implementation",
        "name": "Grounded Implementation & Anti-Invention",
        "purpose": "Enforce grounded implementation discipline: inspect reality before claiming existence, never invent APIs/symbols, and fail closed when authority is missing.",
        "risk_level": "high",
        "modes": ["development", "review"],
        "inputs": ["DirectiveIR", "Dossier", "Active decisions", "Repository source"],
        "outputs": ["Grounded implementation diff", "Reality inspection findings"],
        "capabilities": ["filesystem.read", "filesystem.write"],
        "required_evidence": ["inspection", "test"],
        "stop_conditions": ["Mutations grounded and verified", "Blocked on missing authority"],
        "instructions": """---
name: grounded-implementation
description: Enforces grounded implementation: inspect before claim, anti-invention, and fail-closed missing authority handling.
---
# Grounded Implementation & Anti-Invention

## 1. Inspect Before Claim
Never claim that a file, directory, symbol, function, interface, command, flag, or test exists without first inspecting the repository with tools.

## 2. Never Invent Requirements or APIs
Absence of information is not authorization to invent APIs, generic "best practices", or conventions. When required information is missing, transition: UNKNOWN -> search authoritative source -> resolve -> explicit assumption OR block dependent slice.

## 3. Strict Scope Adherence
Do not perform opportunistic refactoring outside the Task scope. Materialize IN, OUT, and INCIDENTAL boundaries. Never mutate files listed under OUT.

## 4. Fail Closed on Missing Authority
If an action is destructive, security-sensitive, or mutates public contracts and canonical authority is missing, fail closed immediately.
"""
    },
    {
        "id": "implementation-reality-verification",
        "name": "Implementation Reality Verification (Anti-False-Green)",
        "purpose": "Differentiate presence vs reachability vs exercise vs verification; eliminate false-green illusions where dead code passes superficial tests.",
        "risk_level": "high",
        "modes": ["testing", "review"],
        "inputs": ["Source code", "Test suites", "Runtime wiring", "Call sites"],
        "outputs": ["Reachability proof", "Call site inventory", "Verification report"],
        "capabilities": ["filesystem.read", "process.spawn"],
        "required_evidence": ["test", "conformance"],
        "stop_conditions": ["Real reachability verified with evidence", "Dead code / false-green detected"],
        "instructions": """---
name: implementation-reality-verification
description: Eliminates false-greens: verifies code is reachable, exercised, evidenced, and verified beyond shallow unit tests.
---
# Implementation Reality Verification

## 1. Do Not Confuse File Presence with Feature Implementation
Finding a file does not mean the feature works. Verify interfaces, call sites, runtime wiring, lifecycle, error paths, and CLI exposure.

## 2. Eliminate False-Green Oracles
Detect tests that pass because assertions are weak, mocks are used instead of real capabilities, or errors are silently swallowed.

## 3. Terminal State Hierarchy
Derive completion strictly through:
implemented -> reachable -> exercised -> evidenced -> verified -> accepted -> released.
Agents are strictly prohibited from declaring DONE by text output.
"""
    },
    {
        "id": "surface-protocol-conformance",
        "name": "Surface & Protocol Conformance",
        "purpose": "Validate that CLI, TUI, GUI, and daemon share identical domain semantics without bypassing permissions, budgets, or error taxonomy.",
        "risk_level": "high",
        "modes": ["testing", "review"],
        "inputs": ["Protocol schemas", "Client implementations", "Daemon endpoints", "Commands"],
        "outputs": ["Surface conformance report", "Protocol drift matrix"],
        "capabilities": ["process.spawn", "network.client"],
        "required_evidence": ["conformance", "integration"],
        "stop_conditions": ["All surfaces conform to protocol", "Semantic drift detected"],
        "instructions": """---
name: surface-protocol-conformance
description: Guarantees semantic uniformity across CLI, TUI, GUI, and daemon over the public Prumo Protocol.
---
# Surface & Protocol Conformance

## 1. Single Domain Source
A domain rule must exist once in application/domain services. Clients and surfaces must never duplicate domain rules locally or invent private behavior.

## 2. Non-Bypassable Governance
Ensure no surface bypasses the Permission Engine, Budget Manager, or Gauntlet Quality Gates.

## 3. Protocol Version Negotiation & Replay
Verify that clients negotiate protocol versions and can reconnect/replay events without state corruption.
"""
    }
]

# Load existing catalog skills
with open(catalog_skills_file, 'r', encoding='utf-8') as f:
    catalog_data = json.load(f)

existing_skill_ids = {s['id'] for s in catalog_data.get('skills', [])}

for s in p0_skills:
    sid = s['id']
    target_dir = os.path.join(skills_dir, sid)
    os.makedirs(target_dir, exist_ok=True)
    os.makedirs(os.path.join(target_dir, 'checks'), exist_ok=True)
    os.makedirs(os.path.join(target_dir, 'examples'), exist_ok=True)
    
    manifest = {
        "id": s["id"],
        "name": s["name"],
        "version": 1,
        "schema_version": 2,
        "purpose": s["purpose"],
        "risk_level": s["risk_level"],
        "modes": s["modes"],
        "inputs": s["inputs"],
        "outputs": s["outputs"],
        "requires": [],
        "conflicts": [],
        "capabilities": s["capabilities"],
        "references": [],
        "templates": [],
        "checks": [f"checks/{sid}-checklist.md"],
        "scripts": [],
        "examples": ["examples/example.md"],
        "required_evidence": s["required_evidence"],
        "stop_conditions": s["stop_conditions"],
        "provenance": {
            "origin": "framework",
            "license": "MIT",
            "source": "prumo",
            "version": "0.6.0",
            "checksum": "",
            "modified": "2026-09-21"
        },
        "select": {
            "features_any": [sid, "harness"],
            "risk_any": [s["risk_level"]]
        },
        "requires_any": [],
        "tools": [],
        "instructions": s["instructions"]
    }
    
    with open(os.path.join(target_dir, 'manifest.json'), 'w', encoding='utf-8') as mf:
        json.dump(manifest, mf, indent=2, ensure_ascii=False)
        
    with open(os.path.join(target_dir, 'SKILL.md'), 'w', encoding='utf-8') as sf:
        sf.write(s["instructions"])
        
    with open(os.path.join(target_dir, 'checks', f'{sid}-checklist.md'), 'w', encoding='utf-8') as cf:
        cf.write(f"# Checklist for {s['name']}\n\n- [ ] Grounding verified\n- [ ] Inspection complete\n- [ ] Evidence recorded\n")
        
    with open(os.path.join(target_dir, 'examples', 'example.md'), 'w', encoding='utf-8') as ef:
        ef.write(f"# Example for {s['name']}\n\nInspect before claim with tools.\n")
        
    if sid not in existing_skill_ids:
        catalog_data['skills'].append(manifest)
        print(f"Added skill {sid} to catalog.")

with open(catalog_skills_file, 'w', encoding='utf-8') as f:
    json.dump(catalog_data, f, indent=2, ensure_ascii=False)

print(f"Successfully configured P0 skills in {skills_dir} and {catalog_skills_file}.")
