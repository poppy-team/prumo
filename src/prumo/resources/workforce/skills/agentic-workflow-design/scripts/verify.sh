#!/usr/bin/env bash
# Verification script for Agentic Workflow Design.
# Run from the repository root: bash src/prumo/resources/workforce/skills/agentic-workflow-design/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/agentic-workflow-design"

echo "===================================================="
echo " [Prumo Skill: agentic-workflow-design] Agentic Workflow Design Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/agentic-workflow-design-checklist.md" \
         "references/agentic-workflow-design-guide.md" \
         "templates/agentic-workflow-design-spec.md" "scripts/verify.sh"; do
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

if manifest.get("id") != "agentic-workflow-design":
    errors.append("manifest id must be 'agentic-workflow-design'")
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

# 3. Content invariants: agentic-workflow-design domain keywords
echo "--- 3. Checking agentic-workflow-design content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "recipe" "SKILL.md" "SKILL.md covers workflow recipes"
check_grep "handoff" "SKILL.md" "SKILL.md mandates handoff contracts"
check_grep "state transition" "SKILL.md" "SKILL.md mandates explicit state transitions"
check_grep "topology" "SKILL.md" "SKILL.md covers coordination topologies"
check_grep "idempoten" "SKILL.md" "SKILL.md mandates idempotency"
check_grep "delegation depth|depth cap" "SKILL.md" "SKILL.md caps delegation depth"
check_grep "owner|ownership" "checks/agentic-workflow-design-checklist.md" "Checklist gates on single ownership"
check_grep "retry|stall" "checks/agentic-workflow-design-checklist.md" "Checklist covers retries and stall paths"
check_grep "topolog|handoff|idempoten" "references/agentic-workflow-design-guide.md" "Reference guide covers topologies, handoffs, and idempotency"
check_grep "handoff|transition" "templates/agentic-workflow-design-spec.md" "Spec template carries handoff contracts and transitions"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] agentic-workflow-design verification passed cleanly."
  exit 0
else
  echo "[ERROR] agentic-workflow-design verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
