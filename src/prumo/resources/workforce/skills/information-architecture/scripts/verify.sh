#!/usr/bin/env bash
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/information-architecture"

echo "[Prumo Skill: information-architecture] Asset verification"
VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

for file in \
  "SKILL.md" \
  "manifest.json" \
  "references/information-architecture-guide.md" \
  "templates/wireframe-spec.md" \
  "checks/information-architecture-checklist.md" \
  "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$file" ]; then
    pass "non-empty $file"
  else
    fail "missing or empty $file"
  fi
done

if ! command -v python3 >/dev/null 2>&1; then
  fail "python3 is required for manifest validation"
else
  if ! python3 - "$SKILL_DIR" "information-architecture" <<'PY'
import json
import os
import sys
from pathlib import PurePosixPath

skill_dir, expected_id = sys.argv[1], sys.argv[2]
errors = []
with open(os.path.join(skill_dir, "manifest.json"), encoding="utf-8") as handle:
    manifest = json.load(handle)

if manifest.get("id") != expected_id:
    errors.append("manifest id mismatch")
if not isinstance(manifest.get("version"), int) or manifest.get("version", 0) < 2:
    errors.append("version must be at least 2")
if not isinstance(manifest.get("schema_version"), int) or manifest.get("schema_version", 0) < 3:
    errors.append("schema_version must be at least 3")

valid_modes = {"implementation", "review", "audit", "research", "design", "testing", "documentation", "release"}
modes = manifest.get("modes")
if not isinstance(modes, list) or not modes or any(mode not in valid_modes for mode in modes):
    errors.append("modes must contain only supported non-empty values")

for key in ("references", "templates", "checks", "scripts"):
    entries = manifest.get(key)
    if not isinstance(entries, list) or not entries:
        errors.append(f"{key} must be a non-empty array")
        continue
    for relative in entries:
        parts = PurePosixPath(relative).parts if isinstance(relative, str) else ()
        path = os.path.join(skill_dir, relative) if isinstance(relative, str) else ""
        if not parts or ".." in parts or PurePosixPath(relative).is_absolute():
            errors.append(f"unsafe {key} path: {relative!r}")
        elif not os.path.isfile(path) or os.path.getsize(path) == 0:
            errors.append(f"{key} path missing or empty: {relative}")

provenance = manifest.get("provenance", {})
if provenance.get("version") != "0.4.0":
    errors.append("provenance version must be 0.4.0")
if provenance.get("modified") != "2026-09-23":
    errors.append("provenance modified date must be 2026-09-23")

if errors:
    for error in errors:
        print(f"[FAIL] {error}")
    sys.exit(1)
print("[PASS] manifest and wired assets are valid")
PY
  then
    fail "manifest validation failed"
  fi
fi

check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3"
  else
    fail "$3"
  fi
}

check_grep "taxonomy|navigation tree" "SKILL.md" "skill covers taxonomy and navigation structure"
check_grep "progressive disclosure|depth" "SKILL.md" "skill bounds hierarchy and disclosure"
check_grep "tree test|first-click" "SKILL.md" "skill requires wayfinding validation"
check_grep "Inventory Before Hierarchy|Wayfinding" "references/information-architecture-guide.md" "reference defines IA and wayfinding patterns"
check_grep "Navigation Structure|Wayfinding" "checks/information-architecture-checklist.md" "checklist covers hierarchy and validation"
check_grep "Merchant Settings|Navigation Tree" "templates/wireframe-spec.md" "template contains a realistic IA specification"

if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] information-architecture verification passed"
  exit 0
fi

echo "[ERROR] information-architecture verification failed with $VIOLATIONS violation(s)"
exit 1
