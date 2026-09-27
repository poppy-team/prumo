#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="src/prumo/resources/workforce/skills/security-agent-mcp"
echo "[Prumo Skill: security-agent-mcp] Asset verification"
for file in "SKILL.md" "manifest.json" "references/security-agent-mcp-guide.md" "templates/security-agent-mcp-spec.md" "checks/security-agent-mcp-checklist.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    echo "[PASS] $file"
  else
    echo "[FAIL] missing or empty $file"
    exit 1
  fi
done
echo "[SUCCESS] security-agent-mcp verification passed"
exit 0
