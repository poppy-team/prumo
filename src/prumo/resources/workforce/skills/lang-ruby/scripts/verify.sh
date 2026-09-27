#!/bin/sh
set -eu
echo "=== Verifying Ruby Gradual Typing & Defensive Architecture ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-ruby
exit 0
