#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/bytecode-vm-architecture"
echo "[Prumo Skill: bytecode-vm-architecture] Asset verification"
for file in "SKILL.md" "manifest.json" "references/bytecode-vm-architecture-guide.md" "templates/bytecode-vm-architecture-spec.md" "checks/bytecode-vm-architecture-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] bytecode-vm-architecture verification passed"
exit 0
