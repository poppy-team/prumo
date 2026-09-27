#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/binary-security-mitigations"
echo "[Prumo Skill: binary-security-mitigations] Asset verification"
for file in "SKILL.md" "manifest.json" "references/binary-security-mitigations-guide.md" "templates/binary-security-mitigations-spec.md" "checks/binary-security-mitigations-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] binary-security-mitigations verification passed"
exit 0
