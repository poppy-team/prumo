#!/usr/bin/env bash
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Verifying motion-library skill package..."

# Check required files
for file in "manifest.json" "SKILL.md" "checks/motion-library-checklist.md" "templates/motion-spec.md" "references/motion-design-guide.md" "examples/motion-tokens.json"; do
  if [ ! -f "$SKILL_DIR/$file" ]; then
    echo "ERROR: Missing required file: $file"
    exit 1
  fi
done

# Check if manifest is valid JSON
if ! jq empty "$SKILL_DIR/manifest.json" 2>/dev/null; then
  echo "ERROR: manifest.json is not valid JSON."
  exit 1
fi

# Check SKILL.md sections
for section in "Purpose" "Use When" "Do Not Use When" "Required Context" "Procedure" "Decision Rules" "Evidence Required" "Output Contract" "Stop Conditions" "Escalation Rules"; do
  if ! grep -q "#.*$section" "$SKILL_DIR/SKILL.md"; then
    echo "ERROR: SKILL.md missing section: $section"
    exit 1
  fi
done

echo "motion-library skill package verified successfully."
exit 0
