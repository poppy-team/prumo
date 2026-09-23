#!/usr/bin/env bash
# Verification script for API Contract Testing.
# Run from the repository root: bash src/prumo/resources/workforce/skills/api-contract-testing/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/api-contract-testing"

echo "===================================================="
echo " [Prumo Skill: api-contract-testing] Contract Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/api-contract-testing-checklist.md" \
         "references/api-contract-testing-guide.md" \
         "templates/api-contract-testing-spec.md" "scripts/verify.sh"; do
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

if manifest.get("id") != "api-contract-testing":
    errors.append("manifest id must be 'api-contract-testing'")
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

# 3. Content invariants: core api-contract-testing mechanisms
echo "--- 3. Checking api-contract-testing content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "pact|consumer-driven" "SKILL.md" "SKILL.md mandates Pact consumer-driven contracts"
check_grep "openapi" "SKILL.md" "SKILL.md mandates OpenAPI specs"
check_grep "backward compatib|breaking" "SKILL.md" "SKILL.md mandates breaking-change detection"
check_grep "mock|prism" "SKILL.md" "SKILL.md mandates mock servers"
check_grep "breaking|backward compatib" "checks/api-contract-testing-checklist.md" "Checklist covers breaking changes"
check_grep "pact|consumer" "checks/api-contract-testing-checklist.md" "Checklist covers Pact consumers"
check_grep "schema" "checks/api-contract-testing-checklist.md" "Checklist covers schema validation"
check_grep "oasdiff|buf breaking" "references/api-contract-testing-guide.md" "Reference guide covers oasdiff and buf breaking"
check_grep "semver|versioning" "references/api-contract-testing-guide.md" "Reference guide covers semantic versioning"
check_grep "can-i-deploy|verification" "templates/api-contract-testing-spec.md" "Spec template carries can-i-deploy and verification evidence"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] api-contract-testing verification passed cleanly."
  exit 0
else
  echo "[ERROR] api-contract-testing verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
