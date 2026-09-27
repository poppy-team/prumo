#!/usr/bin/env bash
set -e

# verify.sh: Check invariants for the marketing-collateral skill package

# 1. Ensure required directories exist
for dir in checks templates references scripts examples; do
    if [ ! -d "$dir" ]; then
        echo "Error: Directory '$dir' does not exist."
        exit 1
    fi
done

# 2. Check for required files
required_files=(
    "manifest.json"
    "SKILL.md"
    "checks/marketing-collateral-checklist.md"
    "templates/collateral-spec.md"
    "references/marketing-collateral-guide.md"
    "examples/social-media-template.md"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "Error: Required file '$file' is missing."
        exit 1
    fi
done

# 3. Validate SKILL.md sections
sections=(
    "1. Purpose"
    "2. Use When"
    "3. Do Not Use When"
    "4. Required Context"
    "5. Procedure"
    "6. Decision Rules"
    "7. Evidence Required"
    "8. Output Contract"
    "9. Stop Conditions"
    "10. Escalation Rules"
)

for section in "${sections[@]}"; do
    if ! grep -q "$section" SKILL.md; then
        echo "Error: SKILL.md is missing section '$section'."
        exit 1
    fi
done

echo "Marketing collateral skill package verification passed."
exit 0
