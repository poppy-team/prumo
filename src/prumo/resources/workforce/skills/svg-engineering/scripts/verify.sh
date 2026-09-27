#!/usr/bin/env bash
set -e

echo "Verifying SVG Engineering Skill Package invariants..."

# Check directories
for dir in checks templates references examples scripts; do
  if [ ! -d "$dir" ]; then
    echo "Error: Directory $dir does not exist."
    exit 1
  fi
done

# Check required files
for file in manifest.json SKILL.md checks/svg-engineering-checklist.md templates/svg-component-spec.md references/svg-engineering-guide.md examples/accessible-icon.svg; do
  if [ ! -f "$file" ]; then
    echo "Error: File $file does not exist."
    exit 1
  fi
done

# Basic grep checks in SKILL.md
if ! grep -q "viewBox" SKILL.md; then
  echo "Error: SKILL.md is missing viewBox references."
  exit 1
fi

if ! grep -q "currentColor" SKILL.md; then
  echo "Error: SKILL.md is missing currentColor references."
  exit 1
fi

if ! grep -q "SVGO" SKILL.md; then
  echo "Error: SKILL.md is missing SVGO references."
  exit 1
fi

echo "Verification complete. All invariant checks passed."
exit 0
