#!/usr/bin/env bash
# Verification script for user-flows (User Flows & Task Journey Modeling)
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${SKILL_DIR}"

echo "[Prumo Skill: user-flows] Starting verification routine..."

# 1. Mermaid flowchart syntax validation
echo "[1/3] Auditing Mermaid diagrams in specification and examples..."
python3 -c "
import os
import re

files_to_check = ['examples/checkout_flow.md', 'templates/user-flow-spec.md']
for file_path in files_to_check:
    if not os.path.exists(file_path):
        continue
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    mermaid_blocks = re.findall(r'\`\`\`mermaid\s*(.*?)\`\`\`', content, re.DOTALL)
    if not mermaid_blocks:
        raise ValueError(f'No Mermaid flowchart block found in {file_path}')

    for block in mermaid_blocks:
        if 'flowchart' not in block and 'stateDiagram' not in block:
            raise ValueError(f'Mermaid block in {file_path} must declare flowchart or stateDiagram')
        print(f'Mermaid block in {file_path} validated successfully.')
"

# 2. Check for dead-end screens / unhandled forks
echo "[2/3] Checking for decision branch symmetry..."
if [ -f "examples/checkout_flow.md" ]; then
    grep -q -- "-- Yes -->" "examples/checkout_flow.md" || {
        echo "ERROR: Missing positive branch in checkout flow"
        exit 1
    }
    grep -q -- "-- No" "examples/checkout_flow.md" || {
        echo "ERROR: Missing negative branch in checkout flow"
        exit 1
    }
fi

# 3. Verify template validity
echo "[3/3] Verifying user flow deliverable template..."
if [ ! -f "templates/user-flow-spec.md" ]; then
    echo "ERROR: Missing templates/user-flow-spec.md"
    exit 1
fi

echo "[Prumo Skill: user-flows] Verification completed successfully."
exit 0
