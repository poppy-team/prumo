#!/usr/bin/env bash
# Verification script for ux-architecture (Information Architecture & Navigation Strategy)
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${SKILL_DIR}"

echo "[Prumo Skill: ux-architecture] Starting verification routine..."

# 1. Structural graph and sitemap audit
echo "[1/3] Auditing sitemap graph definitions..."
python3 -c '
import os

files = ["examples/sitemap_ia.md", "templates/ia-architecture-spec.md"]
for fpath in files:
    if not os.path.exists(fpath):
        continue
    with open(fpath, "r", encoding="utf-8") as f:
        content = f.read()
    if "mermaid" not in content:
        raise ValueError("Missing Mermaid sitemap graph in " + fpath)
    print("Sitemap graph in " + fpath + " verified.")
'

# 2. Wayfinding accessibility check (breadcrumbs)
echo "[2/3] Checking breadcrumb accessibility semantics..."
if [ -f "examples/sitemap_ia.md" ]; then
    grep -q 'aria-label="Breadcrumb"' "examples/sitemap_ia.md" || {
        echo "ERROR: Breadcrumb missing aria-label='Breadcrumb'"
        exit 1
    }
    grep -q 'aria-current="page"' "examples/sitemap_ia.md" || {
        echo "ERROR: Breadcrumb missing aria-current='page'"
        exit 1
    }
fi

# 3. Template verification
echo "[3/3] Checking IA specification template..."
if [ ! -f "templates/ia-architecture-spec.md" ]; then
    echo "ERROR: Missing templates/ia-architecture-spec.md"
    exit 1
fi

echo "[Prumo Skill: ux-architecture] Verification completed successfully."
exit 0
