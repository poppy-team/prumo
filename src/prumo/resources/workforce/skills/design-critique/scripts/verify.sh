#!/usr/bin/env bash
set -e

# verify.sh for design-critique skill package

# Ensure manifest exists
if [ ! -f "manifest.json" ]; then
    echo "Error: manifest.json is missing."
    exit 1
fi

# Ensure all sections are present in SKILL.md
if ! grep -q "1. Purpose" SKILL.md; then echo "Missing Purpose in SKILL.md"; exit 1; fi
if ! grep -q "2. Use When" SKILL.md; then echo "Missing Use When in SKILL.md"; exit 1; fi
if ! grep -q "3. Do Not Use When" SKILL.md; then echo "Missing Do Not Use When in SKILL.md"; exit 1; fi
if ! grep -q "4. Required Context" SKILL.md; then echo "Missing Required Context in SKILL.md"; exit 1; fi
if ! grep -q "5. Procedure" SKILL.md; then echo "Missing Procedure in SKILL.md"; exit 1; fi
if ! grep -q "6. Decision Rules" SKILL.md; then echo "Missing Decision Rules in SKILL.md"; exit 1; fi
if ! grep -q "7. Evidence Required" SKILL.md; then echo "Missing Evidence Required in SKILL.md"; exit 1; fi
if ! grep -q "8. Output Contract" SKILL.md; then echo "Missing Output Contract in SKILL.md"; exit 1; fi
if ! grep -q "9. Stop Conditions" SKILL.md; then echo "Missing Stop Conditions in SKILL.md"; exit 1; fi
if ! grep -q "10. Escalation Rules" SKILL.md; then echo "Missing Escalation Rules in SKILL.md"; exit 1; fi

echo "Verification passed."
exit 0
