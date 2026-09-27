#!/usr/bin/env bash
# Verification script for Binary & Data Serialization.
# Run from the repository root: bash src/prumo/resources/workforce/skills/serialization/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/serialization"

echo "===================================================="
echo " [Prumo Skill: serialization] Serialization Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/serialization-checklist.md" \
         "references/serialization-guide.md" \
         "templates/serialization-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Manifest validity and asset cross-references
echo "--- 2. Validating manifest and asset cross-references ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" <<'EOF'
import json
import os
import sys

skill_dir = sys.argv[1]
errors = []

with open(os.path.join(skill_dir, "manifest.json"), encoding="utf-8") as fh:
    manifest = json.load(fh)

if manifest.get("id") != "serialization":
    errors.append("manifest id must be 'serialization'")
if manifest.get("version", 0) < 2:
    errors.append("manifest version must be >= 2")
if manifest.get("schema_version", 0) < 3:
    errors.append("manifest schema_version must be >= 3")

valid_modes = {"implementation", "review", "audit", "research",
               "design", "testing", "documentation", "release"}
for mode in manifest.get("modes", []):
    if mode not in valid_modes:
        errors.append(f"invalid mode {mode!r} in manifest")

for key in ("references", "templates", "checks", "scripts"):
    for rel in manifest.get(key, []):
        if not os.path.isfile(os.path.join(skill_dir, rel)):
            errors.append(f"manifest {key} entry missing on disk: {rel}")

if errors:
    for err in errors:
        print(f"[FAIL] {err}")
    sys.exit(1)
print("[PASS] Manifest is valid and all listed assets exist.")
EOF
  then
    fail "Manifest validation failed (see details above)."
  fi
else
  echo "[SKIP] python3 not found; skipping manifest validation."
fi

# 3. Content invariants: core serialization mechanisms
echo "--- 3. Checking serialization content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "protobuf" "SKILL.md" "SKILL.md mandates Protobuf schemas"
check_grep "deterministic" "SKILL.md" "SKILL.md mandates deterministic encoding"
check_grep "round-trip|roundtrip" "SKILL.md" "SKILL.md mandates round-trip guarantees"
check_grep "schema evolution|backward" "SKILL.md" "SKILL.md mandates schema evolution discipline"
check_grep "deterministic|canonical" "checks/serialization-checklist.md" "Checklist covers deterministic canonical encoding"
check_grep "backward|forward compat" "checks/serialization-checklist.md" "Checklist covers compatibility proof"
check_grep "fuzz|round-trip" "checks/serialization-checklist.md" "Checklist covers fuzz round-trip testing"
check_grep "varint|zigzag|tag-length-value" "references/serialization-guide.md" "Reference guide covers varint and TLV encoding"
check_grep "endianness|byte order" "references/serialization-guide.md" "Reference guide covers endianness"
check_grep "messagepack|payload budget|protobuf" "templates/serialization-spec.md" "Spec template carries format and payload budget evidence"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] serialization verification passed cleanly."
  exit 0
else
  echo "[ERROR] serialization verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
