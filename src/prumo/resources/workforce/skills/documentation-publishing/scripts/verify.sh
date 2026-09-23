#!/usr/bin/env bash
# Verification script for Documentation Publishing.
# Run from the repository root: bash src/prumo/resources/workforce/skills/documentation-publishing/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/documentation-publishing"

echo "===================================================="
echo " [Prumo Skill: documentation-publishing] Documentation Publishing Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/documentation-publishing-checklist.md" \
         "references/documentation-publishing-guide.md" \
         "templates/documentation-publishing-spec.md" "scripts/verify.sh"; do
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

if manifest.get("id") != "documentation-publishing":
    errors.append("manifest id must be 'documentation-publishing'")
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

# 3. Content invariants: documentation-publishing domain keywords
echo "--- 3. Checking documentation-publishing content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "single-source|canonical sources" "SKILL.md" "SKILL.md mandates single-source builds"
check_grep "redirect" "SKILL.md" "SKILL.md mandates redirect management"
check_grep "sitemap" "SKILL.md" "SKILL.md mandates sitemap parity"
check_grep "public.*internal|internal.*public|visibility" "SKILL.md" "SKILL.md covers public/internal view separation"
check_grep "version" "SKILL.md" "SKILL.md covers versioned releases"
check_grep "smoke test|HTTP 200" "SKILL.md" "SKILL.md mandates post-deploy smoke tests"
check_grep "orphan|sitemap" "checks/documentation-publishing-checklist.md" "Checklist covers orphans and sitemap"
check_grep "redirect" "checks/documentation-publishing-checklist.md" "Checklist covers redirects"
check_grep "single-source|lychee|sitemap" "references/documentation-publishing-guide.md" "Reference guide covers builds, links, and sitemap"
check_grep "redirect|sitemap|lychee" "templates/documentation-publishing-spec.md" "Release template covers redirects, sitemap, and link evidence"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] documentation-publishing verification passed cleanly."
  exit 0
else
  echo "[ERROR] documentation-publishing verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
