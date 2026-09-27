#!/usr/bin/env bash
# Verification script for Web Performance.
# Run from the repository root: bash src/prumo/resources/workforce/skills/performance-web/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/performance-web"

echo "===================================================="
echo " [Prumo Skill: performance-web] Web Performance Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/performance-web-checklist.md" \
         "references/performance-web-guide.md" \
         "templates/performance-web-spec.md" "scripts/verify.sh"; do
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

if manifest.get("id") != "performance-web":
    errors.append("manifest id must be 'performance-web'")
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

# 3. Content invariants: core performance-web mechanisms
echo "--- 3. Checking performance-web content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "lcp|core web vitals" "SKILL.md" "SKILL.md mandates LCP and Core Web Vitals"
check_grep "lighthouse" "SKILL.md" "SKILL.md mandates Lighthouse budgets"
check_grep "bundle" "SKILL.md" "SKILL.md mandates bundle budgets"
check_grep "waterfall" "SKILL.md" "SKILL.md mandates waterfall analysis"
check_grep "lcp|inp|cls" "checks/performance-web-checklist.md" "Checklist covers LCP, INP, and CLS"
check_grep "bundle|budget" "checks/performance-web-checklist.md" "Checklist covers bundle budgets"
check_grep "image|font" "checks/performance-web-checklist.md" "Checklist covers images and fonts"
check_grep "ttfb|fcp|lcp|inp|cls" "references/performance-web-guide.md" "Reference guide covers web performance metrics"
check_grep "critical render" "references/performance-web-guide.md" "Reference guide covers critical render path"
check_grep "lighthouse|rum" "templates/performance-web-spec.md" "Spec template carries Lighthouse and RUM evidence"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] performance-web verification passed cleanly."
  exit 0
else
  echo "[ERROR] performance-web verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
