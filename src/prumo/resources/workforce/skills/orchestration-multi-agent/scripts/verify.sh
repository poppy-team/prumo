#!/usr/bin/env bash
# Verification script for Multi-Agent Orchestration.
# Run from the repository root: bash src/prumo/resources/workforce/skills/orchestration-multi-agent/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/orchestration-multi-agent"

echo "===================================================="
echo " [Prumo Skill: orchestration-multi-agent] Multi-Agent Orchestration Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/orchestration-multi-agent-checklist.md" \
         "references/orchestration-multi-agent-guide.md" \
         "templates/orchestration-multi-agent-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Manifest validity and asset cross-references
echo "--- 2. Validating manifest and asset cross-references ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" <<'EOF'
import json
import os
import sys

skill_dir = sys.argv[1]
errors = []

with open(os.path.join(skill_dir, "manifest.json"), encoding="utf-8") as fh:
    manifest = json.load(fh)

if manifest.get("id") != "orchestration-multi-agent":
    errors.append("manifest id must be 'orchestration-multi-agent'")
if manifest.get("version", 0) < 2:
    errors.append("manifest version must be >= 2")
if manifest.get("schema_version", 0) < 3:
    errors.append("manifest schema_version must be >= 3")

valid_modes = {"implementation", "review", "audit", "research",
               "design", "testing", "documentation", "release"}
for mode in manifest.get("modes", []):
    if mode not in valid_modes:
        errors.append(f"invalid mode {mode!r} in manifest")

for key in ("references", "templates", "checks", "scripts"):
    for rel in manifest.get(key, []):
        if not os.path.isfile(os.path.join(skill_dir, rel)):
            errors.append(f"manifest {key} entry missing on disk: {rel}")

if errors:
    for err in errors:
        print(f"[FAIL] {err}")
    sys.exit(1)
print("[PASS] Manifest is valid and all listed assets exist.")
EOF
  then
    fail "Manifest validation failed (see details above)."
  fi
else
  echo "[SKIP] python3 not found; skipping manifest validation."
fi

# 3. Content invariants: orchestration-multi-agent domain keywords
echo "--- 3. Checking orchestration-multi-agent content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "DAG|depends-on|critical path" "SKILL.md" "SKILL.md mandates DAG planning"
check_grep "isolat|worktree" "SKILL.md" "SKILL.md mandates worker isolation"
check_grep "retr(y|ies).*cap|cap.*retr" "SKILL.md" "SKILL.md caps retries"
check_grep "delegation depth|depth.*cap" "SKILL.md" "SKILL.md caps delegation depth"
check_grep "reviewer|author != reviewer" "SKILL.md" "SKILL.md mandates reviewer diversity"
check_grep "evidence.*gate|gate.*evidence|CI.*arbiter|final arbiter" "SKILL.md" "SKILL.md gates completion on evidence and CI"
check_grep "allowlist|worktree" "checks/orchestration-multi-agent-checklist.md" "Checklist covers isolation mechanics"
check_grep "caps ledger|parked" "checks/orchestration-multi-agent-checklist.md" "Checklist covers caps ledger and parking"
check_grep "DAG|isolation|reviewer diversity|evidence" "references/orchestration-multi-agent-guide.md" "Reference guide covers DAG, isolation, reviewers, and evidence"
check_grep "DAG|reviewer|CI green" "templates/orchestration-multi-agent-spec.md" "Spec template carries DAG record, reviewers, and CI verdicts"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] orchestration-multi-agent verification passed cleanly."
  exit 0
else
  echo "[ERROR] orchestration-multi-agent verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
