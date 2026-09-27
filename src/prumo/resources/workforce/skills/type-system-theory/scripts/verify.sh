#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/type-system-theory"
echo "[Prumo Skill: type-system-theory] Asset verification"
for file in "SKILL.md" "manifest.json" "references/type-system-theory-guide.md" "templates/type-system-theory-spec.md" "checks/type-system-theory-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] type-system-theory verification passed"
exit 0
