#!/usr/bin/env bash
# verify.sh for advertising-design skill

set -e

echo "Verifying advertising-design skill package..."

SKILL_DIR="$(dirname "$0")/.."

# Check required files
for file in manifest.json SKILL.md checks/advertising-design-checklist.md templates/ad-design-spec.md references/advertising-design-guide.md examples/aida-campaign-example.md; do
  if [ ! -f "$SKILL_DIR/$file" ]; then
    echo "Error: Missing $file"
    exit 1
  fi
done

# Validate manifest JSON
if ! jq empty "$SKILL_DIR/manifest.json" 2>/dev/null; then
    echo "Error: manifest.json is invalid JSON"
    exit 1
fi

echo "advertising-design skill package verified successfully."
exit 0
