#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/computational-physics"
echo "[Prumo Skill: computational-physics] Asset verification"
for file in "SKILL.md" "manifest.json" "references/computational-physics-guide.md" "templates/computational-physics-spec.md" "checks/computational-physics-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] computational-physics verification passed"
exit 0
