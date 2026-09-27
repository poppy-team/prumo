#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/realtime-audio-dsp"
echo "[Prumo Skill: realtime-audio-dsp] Asset verification"
for file in "SKILL.md" "manifest.json" "references/realtime-audio-dsp-guide.md" "templates/realtime-audio-dsp-spec.md" "checks/realtime-audio-dsp-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] realtime-audio-dsp verification passed"
exit 0
