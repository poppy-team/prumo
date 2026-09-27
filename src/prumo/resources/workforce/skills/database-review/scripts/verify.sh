#!/usr/bin/env bash
# Verification script for Database & Schema Review.
# Run from the repository root: bash src/prumo/resources/workforce/skills/database-review/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/database-review"

echo "===================================================="
echo " [Prumo Skill: database-review] Database Review Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/database-review-checklist.md" \
         "references/database-review-guide.md" \
         "templates/database-review-spec.md" "scripts/verify.sh"; do
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

if manifest.get("id") != "database-review":
    errors.append("manifest id must be 'database-review'")
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

# 3. Content invariants: core database-review mechanisms
echo "--- 3. Checking database-review content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "explain" "SKILL.md" "SKILL.md mandates EXPLAIN ANALYZE evidence"
check_grep "index" "SKILL.md" "SKILL.md mandates index review"
check_grep "migration" "SKILL.md" "SKILL.md mandates migration review"
check_grep "transaction" "SKILL.md" "SKILL.md mandates transaction review"
check_grep "sequential scan|seq scan" "checks/database-review-checklist.md" "Checklist covers sequential scan detection"
check_grep "foreign key" "checks/database-review-checklist.md" "Checklist covers foreign key constraints"
check_grep "migration|rollback" "checks/database-review-checklist.md" "Checklist covers migration rollback"
check_grep "b-tree|btree|gin|gist" "references/database-review-guide.md" "Reference guide covers index internals"
check_grep "isolation|serializable" "references/database-review-guide.md" "Reference guide covers isolation levels"
check_grep "pg_stat|slow query" "templates/database-review-spec.md" "Spec template carries pg_stat and slow query evidence"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] database-review verification passed cleanly."
  exit 0
else
  echo "[ERROR] database-review verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
