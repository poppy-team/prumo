#!/bin/bash
# verify.sh for design-psychology skill
# Usage: ./verify.sh <path-to-audit-report.md>

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <path-to-audit-report.md>"
  exit 1
fi

REPORT_FILE="$1"

if [ ! -f "$REPORT_FILE" ]; then
  echo "Error: Audit report file '$REPORT_FILE' not found."
  exit 1
fi

echo "Verifying Cognitive Audit Report: $REPORT_FILE"

# Check for required sections based on templates/cognitive-audit-report.md
REQUIRED_SECTIONS=(
  "Hick's Law"
  "Fitts's Law"
  "Gestalt"
  "Cognitive Load"
)

MISSING=0
for section in "${REQUIRED_SECTIONS[@]}"; do
  if ! grep -qi "$section" "$REPORT_FILE"; then
    echo "❌ Missing section or keyword: $section"
    MISSING=1
  else
    echo "✅ Found: $section"
  fi
done

if [ $MISSING -eq 1 ]; then
  echo "Verification failed: The audit report is incomplete."
  exit 1
fi

echo "✅ Verification passed: The audit report contains all required psychological evaluations."
exit 0
