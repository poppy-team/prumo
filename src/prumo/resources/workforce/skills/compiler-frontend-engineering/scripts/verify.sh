#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/compiler-frontend-engineering"
echo "[Prumo Skill: compiler-frontend-engineering] Asset verification"
for file in "SKILL.md" "manifest.json" "references/compiler-frontend-engineering-guide.md" "templates/compiler-frontend-engineering-spec.md" "checks/compiler-frontend-engineering-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] compiler-frontend-engineering verification passed"
exit 0
