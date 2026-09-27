#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/security-saas-isolation"
echo "[Prumo Skill: security-saas-isolation] Asset verification"
for file in "SKILL.md" "manifest.json" "references/security-saas-isolation-guide.md" "templates/security-saas-isolation-spec.md" "checks/security-saas-isolation-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] security-saas-isolation verification passed"
exit 0
