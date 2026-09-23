#!/usr/bin/env bash
# Verification script for Documentation for LLM Consumption.
# Run from the repository root: bash src/prumo/resources/workforce/skills/documentation-for-llms/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/documentation-for-llms"

echo "===================================================="
echo " [Prumo Skill: documentation-for-llms] Documentation for LLM Consumption Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/documentation-for-llms-checklist.md" \
         "references/documentation-for-llms-guide.md" \
         "templates/documentation-for-llms-spec.md" "scripts/verify.sh"; do
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

if manifest.get("id") != "documentation-for-llms":
    errors.append("manifest id must be 'documentation-for-llms'")
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

# 3. Content invariants: LLM-documentation domain keywords
echo "--- 3. Checking documentation-for-llms content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "progressive disclosure" "SKILL.md" "SKILL.md mandates progressive disclosure"
check_grep "llms\.txt" "SKILL.md" "SKILL.md mandates llms.txt indexing"
check_grep "frontmatter" "SKILL.md" "SKILL.md mandates frontmatter metadata"
check_grep "anchor" "SKILL.md" "SKILL.md mandates stable anchors"
check_grep "chunk" "SKILL.md" "SKILL.md mandates chunk-size discipline"
check_grep "single source of truth" "SKILL.md" "SKILL.md mandates single source of truth"
check_grep "markdownlint" "checks/documentation-for-llms-checklist.md" "Checklist gates on markdownlint"
check_grep "lychee|link-check|broken links" "checks/documentation-for-llms-checklist.md" "Checklist gates on link checking"
check_grep "llmstxt|frontmatter schema|chunk" "references/documentation-for-llms-guide.md" "Reference guide covers index, schema, and chunking"
check_grep "canonical|token" "templates/documentation-for-llms-spec.md" "Spec template covers canonical map and token targets"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] documentation-for-llms verification passed cleanly."
  exit 0
else
  echo "[ERROR] documentation-for-llms verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
