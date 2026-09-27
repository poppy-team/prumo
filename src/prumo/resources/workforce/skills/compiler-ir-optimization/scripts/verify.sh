#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/compiler-ir-optimization"
echo "[Prumo Skill: compiler-ir-optimization] Asset verification"
for file in "SKILL.md" "manifest.json" "references/compiler-ir-optimization-guide.md" "templates/compiler-ir-optimization-spec.md" "checks/compiler-ir-optimization-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] compiler-ir-optimization verification passed"
exit 0
