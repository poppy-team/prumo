#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/memory-safety-sanitizers"
echo "[Prumo Skill: memory-safety-sanitizers] Asset verification"
for file in "SKILL.md" "manifest.json" "references/memory-safety-sanitizers-guide.md" "templates/memory-safety-sanitizers-spec.md" "checks/memory-safety-sanitizers-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] memory-safety-sanitizers verification passed"
exit 0
