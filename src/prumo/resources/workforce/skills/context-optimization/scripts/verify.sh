#!/usr/bin/env bash
# Verification script for Context Optimization.
# Run from the repository root: bash src/prumo/resources/workforce/skills/context-optimization/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/context-optimization"

echo "===================================================="
echo " [Prumo Skill: context-optimization] Context Optimization Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/context-optimization-checklist.md" \
         "references/context-optimization-guide.md" \
         "templates/context-optimization-spec.md" "scripts/verify.sh"; do
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

if manifest.get("id") != "context-optimization":
    errors.append("manifest id must be 'context-optimization'")
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

# 3. Content invariants: context-optimization domain keywords
echo "--- 3. Checking context-optimization content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "dedup" "SKILL.md" "SKILL.md mandates deduplication"
check_grep "prun" "SKILL.md" "SKILL.md mandates relevance pruning"
check_grep "summariz" "SKILL.md" "SKILL.md covers summarization of verbose payloads"
check_grep "verbatim" "SKILL.md" "SKILL.md protects critical facts verbatim"
check_grep "cache" "SKILL.md" "SKILL.md covers cache-directive annotations"
check_grep "tokens per.*goal|token accounting" "SKILL.md" "SKILL.md mandates token accounting"
check_grep "verbatim|fidelity" "checks/context-optimization-checklist.md" "Checklist guards verbatim fidelity"
check_grep "cache|hit rate" "checks/context-optimization-checklist.md" "Checklist covers cache discipline"
check_grep "relevance tier|prompt caching|tokens per" "references/context-optimization-guide.md" "Reference guide covers tiers, caching, and the goal metric"
check_grep "accounting|verbatim|cache" "templates/context-optimization-spec.md" "Spec template carries accounting, fidelity, and cache records"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] context-optimization verification passed cleanly."
  exit 0
else
  echo "[ERROR] context-optimization verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
