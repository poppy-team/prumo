#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/security-authz-matrix"
echo "[Prumo Skill: security-authz-matrix] Asset verification"
for file in "SKILL.md" "manifest.json" "references/security-authz-matrix-guide.md" "templates/security-authz-matrix-spec.md" "checks/security-authz-matrix-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] security-authz-matrix verification passed"
exit 0
