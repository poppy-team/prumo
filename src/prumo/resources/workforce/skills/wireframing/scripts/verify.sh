#!/usr/bin/env bash
set -euo pipefail

TARGET_FILE="${1:-src/prumo/resources/workforce/skills/wireframing/examples/dashboard_wireframe.html}"

if [ ! -f "$TARGET_FILE" ]; then
    echo "Target wireframe file not found: $TARGET_FILE" >&2
    exit 1
fi

echo "Verifying wireframe fidelity, semantic landmarks and structure for: $TARGET_FILE"

python3 - <<PYCHECK
import sys
import re

path = "$TARGET_FILE"
with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

required_landmarks = ["<header", "<nav", "<main", "<footer"]
missing_landmarks = [lm for lm in required_landmarks if lm not in content]

if missing_landmarks:
    print(f"Error: Missing semantic HTML5 landmarks in wireframe: {missing_landmarks}", file=sys.stderr)
    sys.exit(1)

if "viewport" not in content.lower():
    print("Error: Wireframe lacks responsive viewport meta configuration", file=sys.stderr)
    sys.exit(1)

# Check for grid or flex layout cues
if not re.search(r"(display:\s*(grid|flex)|grid-template-columns)", content, re.IGNORECASE):
    print("Warning: Wireframe lacks explicit CSS grid/flex layout specifications", file=sys.stderr)

print("Semantic landmarks, viewport and responsive structure verified successfully.")
PYCHECK

echo "Wireframing verification passed."
exit 0
