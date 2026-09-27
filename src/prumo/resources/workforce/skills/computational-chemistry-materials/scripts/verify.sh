#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/computational-chemistry-materials"
echo "[Prumo Skill: computational-chemistry-materials] Asset verification"
for file in "SKILL.md" "manifest.json" "references/computational-chemistry-materials-guide.md" "templates/computational-chemistry-materials-spec.md" "checks/computational-chemistry-materials-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] computational-chemistry-materials verification passed"
exit 0
