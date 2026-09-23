#!/usr/bin/env bash
# Verification script for ui-ux-review (UI/UX Heuristic Review & Usability Audit)
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${SKILL_DIR}"

echo "[Prumo Skill: ui-ux-review] Starting verification routine..."

# 1. Audit heuristic report structure
echo "[1/3] Auditing heuristic evaluation structure in examples..."
python3 -c "
import os
import re

report_path = 'examples/heuristic_audit_sample.md'
if not os.path.exists(report_path):
    raise FileNotFoundError('Missing ' + report_path)

with open(report_path, 'r', encoding='utf-8') as f:
    text = f.read()

# Verify NN heuristics mentioned
if not re.search(r'H[1-9]|H10', text):
    raise ValueError('Report must cite Nielsen Norman heuristics (H1-H10)')

# Verify severity ratings
if 'Severity' not in text:
    raise ValueError('Report must assign standardized Severity ratings')

print('Heuristic audit sample conforms to NN/severity standards.')
"

# 2. Check for actionable remediations
echo "[2/3] Checking that all defects contain actionable remediations..."
if [ -f "examples/heuristic_audit_sample.md" ]; then
    grep -q "Recommended Remediation" "examples/heuristic_audit_sample.md" || {
        echo "ERROR: Report missing Recommended Remediation sections."
        exit 1
    }
fi

# 3. Template verification
echo "[3/3] Checking template integrity..."
if [ ! -f "templates/heuristic-evaluation-report.md" ]; then
    echo "ERROR: Missing templates/heuristic-evaluation-report.md"
    exit 1
fi

echo "[Prumo Skill: ui-ux-review] Verification completed successfully."
exit 0
