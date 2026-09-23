#!/usr/bin/env bash
# Verification script for Prompt Engineering & Evaluation.
# Run from the repository root: bash src/prumo/resources/workforce/skills/prompt-engineering/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/prompt-engineering"

echo "===================================================="
echo " [Prumo Skill: prompt-engineering] Prompt Engineering & Evaluation Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/prompt-engineering-checklist.md" \
         "references/prompt-engineering-guide.md" \
         "templates/prompt-engineering-spec.md" "scripts/verify.sh"; do
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

if manifest.get("id") != "prompt-engineering":
    errors.append("manifest id must be 'prompt-engineering'")
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

# 3. Content invariants: prompt-engineering domain keywords
echo "--- 3. Checking prompt-engineering content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "system prompt" "SKILL.md" "SKILL.md covers system prompt authoring"
check_grep "few-shot" "SKILL.md" "SKILL.md covers few-shot examples"
check_grep "eval harness|eval set" "SKILL.md" "SKILL.md mandates an eval harness"
check_grep "regression" "SKILL.md" "SKILL.md mandates regression gates"
check_grep "temperature|sampling" "SKILL.md" "SKILL.md mandates pinned sampling config"
check_grep "version" "SKILL.md" "SKILL.md mandates prompt versioning"
check_grep "seed|baseline" "checks/prompt-engineering-checklist.md" "Checklist gates on seeded baselines"
check_grep "synthetic|secrets" "checks/prompt-engineering-checklist.md" "Checklist guards example data hygiene"
check_grep "few-shot|sampling|regression" "references/prompt-engineering-guide.md" "Reference guide covers few-shot, sampling, and regression bars"
check_grep "baseline|candidate|rollback" "templates/prompt-engineering-spec.md" "Spec template carries baseline, candidate, and rollback records"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] prompt-engineering verification passed cleanly."
  exit 0
else
  echo "[ERROR] prompt-engineering verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
