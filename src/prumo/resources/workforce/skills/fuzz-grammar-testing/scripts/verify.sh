#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/fuzz-grammar-testing"
echo "[Prumo Skill: fuzz-grammar-testing] Asset verification"
for file in "SKILL.md" "manifest.json" "references/fuzz-grammar-testing-guide.md" "templates/fuzz-grammar-testing-spec.md" "checks/fuzz-grammar-testing-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] fuzz-grammar-testing verification passed"
exit 0
