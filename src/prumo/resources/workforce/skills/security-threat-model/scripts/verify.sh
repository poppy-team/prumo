#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/security-threat-model"
echo "[Prumo Skill: security-threat-model] Asset verification"
for file in "SKILL.md" "manifest.json" "references/security-threat-model-guide.md" "templates/security-threat-model-spec.md" "checks/security-threat-model-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] security-threat-model verification passed"
exit 0
