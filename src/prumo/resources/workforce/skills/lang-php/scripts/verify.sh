#!/bin/sh
set -eu
echo "=== Verifying PHP Strict Types & Maximum Static Analysis ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-php
exit 0
