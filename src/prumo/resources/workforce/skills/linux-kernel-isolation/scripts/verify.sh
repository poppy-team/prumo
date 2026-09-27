#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/linux-kernel-isolation"
echo "[Prumo Skill: linux-kernel-isolation] Asset verification"
for file in "SKILL.md" "manifest.json" "references/linux-kernel-isolation-guide.md" "templates/linux-kernel-isolation-spec.md" "checks/linux-kernel-isolation-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] linux-kernel-isolation verification passed"
exit 0
