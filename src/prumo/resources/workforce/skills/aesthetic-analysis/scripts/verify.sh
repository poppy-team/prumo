#!/bin/bash
# Verify the aesthetic-analysis skill package structure

set -e

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Verifying aesthetic-analysis package..."

# Check required files exist
REQUIRED_FILES=(
  "manifest.json"
  "SKILL.md"
  "checks/aesthetic-analysis-checklist.md"
  "templates/aesthetic-evaluation.md"
  "references/aesthetic-analysis-guide.md"
  "examples/aesthetic-comparison.md"
)

for file in "${REQUIRED_FILES[@]}"; do
  if [ ! -f "$SKILL_DIR/$file" ]; then
    echo "ERROR: Missing required file: $file"
    exit 1
  fi
done

# Check jq is installed
if ! command -v jq &> /dev/null; then
  echo "WARNING: jq not found, skipping manifest JSON validation."
else
  # Verify manifest schema version
  SCHEMA_VER=$(jq -r '.schema_version' "$SKILL_DIR/manifest.json")
  if [ "$SCHEMA_VER" != "3" ]; then
    echo "ERROR: manifest.json schema_version must be 3, found: $SCHEMA_VER"
    exit 1
  fi
fi

# Check SKILL.md sections
SECTIONS=("1. Purpose" "2. Use When" "3. Do Not Use When" "4. Required Context" "5. Procedure" "6. Decision Rules" "7. Evidence Required" "8. Output Contract" "9. Stop Conditions" "10. Escalation Rules")
for section in "${SECTIONS[@]}"; do
  if ! grep -q "$section" "$SKILL_DIR/SKILL.md"; then
    echo "ERROR: SKILL.md is missing section: $section"
    exit 1
  fi
done

echo "aesthetic-analysis package verification passed."
exit 0
