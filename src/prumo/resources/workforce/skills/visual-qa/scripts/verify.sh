#!/usr/bin/env bash
# Verification script for visual-qa (Visual Quality Assurance)
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${SKILL_DIR}"

echo "[Prumo Skill: visual-qa] Starting verification routine..."

# 1. Audit sample visual QA scorecard
echo "[1/3] Auditing visual QA scorecard structure in examples..."
python3 -c "
import os

sample_path = 'examples/visual_qa_audit_sample.md'
if not os.path.exists(sample_path):
    raise FileNotFoundError('Missing ' + sample_path)

with open(sample_path, 'r', encoding='utf-8') as f:
    text = f.read()

required_terms = ['Spatial Rhythm', 'Typography Scale', 'Color & Elevation', 'Breakpoint Inspection']
for term in required_terms:
    if term not in text:
        raise ValueError('Sample report missing required dimension: ' + term)

print('Visual QA audit sample validated successfully.')
"

# 2. Template verification
echo "[2/3] Checking visual QA report template..."
if [ ! -f "templates/visual-qa-report.md" ]; then
    echo "ERROR: Missing templates/visual-qa-report.md"
    exit 1
fi

# 3. Off-grid spacing scanner in example styles (detects odd pixel values like 11px, 13px)
echo "[3/3] Auditing for off-grid spacing values..."
OFF_GRID_MATCHES="$(
  find "${SKILL_DIR}" -type f \( -name "*.css" -o -name "*.scss" \) \
    -exec grep -HnE 'margin.*:\s*([13579]|1[13579]|2[1357])px' {} + 2>/dev/null || true
)"

if [ -n "$OFF_GRID_MATCHES" ]; then
    echo "WARNING: Odd pixel off-grid spacing discovered (snap to 4px/8px tokens):"
    echo "$OFF_GRID_MATCHES" | head -n 5
fi

echo "[Prumo Skill: visual-qa] Verification completed successfully."
exit 0
