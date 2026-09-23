#!/usr/bin/env bash
# Verification script for Playwright UI Testing.
# Run from the repository root: bash src/prumo/resources/workforce/skills/playwright-ui/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/playwright-ui"

echo "===================================================="
echo " [Prumo Skill: playwright-ui] Playwright UI Testing Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/playwright-ui-checklist.md" \
         "references/playwright-ui-guide.md" \
         "templates/playwright-ui-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Manifest validity and asset cross-references
echo "--- 2. Validating manifest and asset cross-references ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" "playwright-ui" <<'EOF'
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

# 3. Content invariants: selectors, fixtures, browsers, evidence
echo "--- 3. Checking playwright-ui content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "getByRole|semantic" "SKILL.md" "SKILL.md mandates semantic selectors"
check_grep "fixture|storageState" "SKILL.md" "SKILL.md mandates fixtures and auth state"
check_grep "shard|chromium.*firefox.*webkit|three browsers" "SKILL.md" "SKILL.md mandates cross-browser sharded runs"
check_grep "trace|screenshot|video" "SKILL.md" "SKILL.md mandates failure evidence"
check_grep "flake|quarantine|retr" "checks/playwright-ui-checklist.md" "Checklist covers flake budget and quarantine"
check_grep "strict mode|auto-wait|web-first" "references/playwright-ui-guide.md" "Reference guide covers strict mode and auto-waiting"
check_grep "Trace Viewer|trace.zip" "references/playwright-ui-guide.md" "Reference guide covers trace debugging"
check_grep "getByRole|getByTestId" "references/playwright-ui-guide.md" "Reference guide shows semantic locator code"
check_grep "chromium|firefox|webkit" "templates/playwright-ui-spec.md" "UI spec template records browser matrix"
check_grep "flake" "templates/playwright-ui-spec.md" "UI spec template records stability metrics"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] playwright-ui verification passed cleanly."
  exit 0
else
  echo "[ERROR] playwright-ui verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
