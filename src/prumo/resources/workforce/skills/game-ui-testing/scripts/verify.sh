#!/usr/bin/env bash
# Verification script for Game UI Testing.
# Run from the repository root: bash src/prumo/resources/workforce/skills/game-ui-testing/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/game-ui-testing"

echo "===================================================="
echo " [Prumo Skill: game-ui-testing] Game UI Testing Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/game-ui-testing-checklist.md" \
         "references/game-ui-testing-guide.md" \
         "templates/game-ui-testing-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Manifest validity and asset cross-references
echo "--- 2. Validating manifest and asset cross-references ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" "game-ui-testing" <<'EOF'
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
print("[PASS] Manifest is valid and all listed assets exist on disk.")
EOF
  then
    fail "Manifest validation failed (see details above)."
  fi
else
  echo "[SKIP] python3 not found; skipping manifest validation."
fi

# 3. Content invariants: navigation, safe areas, localization, transitions
echo "--- 3. Checking game-ui-testing content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "focus" "SKILL.md" "SKILL.md mandates focus ownership"
check_grep "safe.area" "SKILL.md" "SKILL.md mandates safe-area compliance"
check_grep "locali" "SKILL.md" "SKILL.md mandates localization passes"
check_grep "debounce|double.activation" "SKILL.md" "SKILL.md mandates single-fire transitions"
check_grep "focus" "checks/game-ui-testing-checklist.md" "Checklist covers focus navigation"
check_grep "safe.area|action-safe" "checks/game-ui-testing-checklist.md" "Checklist covers safe areas"
check_grep "locali|pseudo|RTL" "checks/game-ui-testing-checklist.md" "Checklist covers localization"
check_grep "anchor|nine-patch|container" "references/game-ui-testing-guide.md" "Reference guide covers resolution-independent layout"
check_grep "pseudo|kinsoku|RTL" "references/game-ui-testing-guide.md" "Reference guide covers text expansion and RTL"
check_grep "resolution|locale" "templates/game-ui-testing-spec.md" "Test spec template covers resolutions and locales"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] game-ui-testing verification passed cleanly."
  exit 0
else
  echo "[ERROR] game-ui-testing verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
