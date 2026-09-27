#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/runtime-memory-gc"
echo "[Prumo Skill: runtime-memory-gc] Asset verification"
for file in "SKILL.md" "manifest.json" "references/runtime-memory-gc-guide.md" "templates/runtime-memory-gc-spec.md" "checks/runtime-memory-gc-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] runtime-memory-gc verification passed"
exit 0
