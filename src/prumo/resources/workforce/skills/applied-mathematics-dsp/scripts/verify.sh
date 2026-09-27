#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/applied-mathematics-dsp"
echo "[Prumo Skill: applied-mathematics-dsp] Asset verification"
for file in "SKILL.md" "manifest.json" "references/applied-mathematics-dsp-guide.md" "templates/applied-mathematics-dsp-spec.md" "checks/applied-mathematics-dsp-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] applied-mathematics-dsp verification passed"
exit 0
