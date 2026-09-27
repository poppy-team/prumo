#!/bin/sh
set -eu
echo "=== Verifying Swift 6 Strict Concurrency & Value Semantics ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-swift
exit 0
