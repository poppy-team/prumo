#!/usr/bin/env bash
set -e

echo "Verifying Icon System Package..."

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Verify required files exist
REQUIRED_FILES=(
  "manifest.json"
  "SKILL.md"
  "checks/icon-system-checklist.md"
  "templates/icon-spec.md"
  "references/icon-system-guide.md"
  "examples/icon-grid-spec.md"
)

for file in "${REQUIRED_FILES[@]}"; do
  if [ ! -f "$DIR/$file" ]; then
    echo "❌ Error: Missing required file: $file"
    exit 1
  fi
done

# Basic JSON validation for manifest
if ! command -v jq &> /dev/null; then
    echo "⚠️ jq not found, skipping manifest syntax check."
else
    if ! jq . "$DIR/manifest.json" > /dev/null; then
        echo "❌ Error: manifest.json is invalid JSON."
        exit 1
    fi
fi

echo "✅ Icon System Package verified successfully!"
exit 0
