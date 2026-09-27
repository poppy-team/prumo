#!/bin/sh
set -eu
echo "=== Verifying Lua Scope Protection & Sandboxing ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-lua
exit 0
