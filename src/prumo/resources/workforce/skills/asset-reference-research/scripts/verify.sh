#!/usr/bin/env bash
# Verification script for Asset Reference Research.
# Run from the repository root: bash src/prumo/resources/workforce/skills/asset-reference-research/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/asset-reference-research"
SKILL_ID="asset-reference-research"

echo "===================================================="
echo " [Prumo Skill: $SKILL_ID] Asset Reference Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/asset-reference-research-checklist.md" \
         "references/asset-reference-research-guide.md" \
         "templates/asset-reference-research-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done
if [ -x "$SKILL_DIR/scripts/verify.sh" ]; then
  pass "verify.sh is executable."
else
  fail "verify.sh is not executable."
fi

# 2. Stale superseded artifacts must be gone
echo "--- 2. Checking for stale superseded artifacts ---"
for f in "templates/asset-reference-research-template.md"; do
  if [ -e "$SKILL_DIR/$f" ]; then
    fail "Stale artifact still present: $SKILL_DIR/$f."
  else
    pass "Stale artifact absent: $f."
  fi
done

# 3. Manifest validity and asset cross-references
echo "--- 3. Validating manifest and asset wiring ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" "$SKILL_ID" <<'EOF'
import json
import os
import sys

skill_dir, skill_id = sys.argv[1], sys.argv[2]
errors = []

with open(os.path.join(skill_dir, "manifest.json"), encoding="utf-8") as fh:
    manifest = json.load(fh)

if manifest.get("id") != skill_id:
    errors.append(f"manifest id must be {skill_id!r}")
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
print("[PASS] Manifest is valid and all wired assets exist.")
EOF
  then
    fail "Manifest validation failed (see details above)."
  fi
else
  echo "[SKIP] python3 not found; skipping manifest validation."
fi

# 4. Content invariants for asset reference research
echo "--- 4. Checking asset-reference-research content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "license matrix|licensing metadata" "SKILL.md" "SKILL.md mandates license matrices"
check_grep "CC0|CC-BY|attribution" "SKILL.md" "SKILL.md mandates license classes and attribution"
check_grep "NC|ND|forbidden|ripped" "SKILL.md" "SKILL.md mandates forbidden-license rules"
check_grep "placeholder|grey-?box|mapping" "SKILL.md" "SKILL.md mandates placeholder mapping"
check_grep "CREDITS|vendor|SHA-256|hash" "SKILL.md" "SKILL.md mandates attribution pack with hashes"
check_grep "OpenGameArt|Kenney|Poly Pizza|Freesound|Unsplash" "references/asset-reference-research-guide.md" "Reference guide covers trusted sources"
check_grep "share-alike|quarantine|OFL|proof" "references/asset-reference-research-guide.md" "Reference guide covers license triage"
check_grep "coverage|forbidden|CREDITS|vendor" "templates/asset-reference-research-spec.md" "Spec template records coverage and attribution evidence"
check_grep "orphan|glyph|loop|pivot" "checks/asset-reference-research-checklist.md" "Checklist covers technical fit checks"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] $SKILL_ID verification passed cleanly."
  exit 0
else
  echo "[ERROR] $SKILL_ID verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
