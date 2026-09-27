#!/bin/sh
set -eu
echo "=== Verifying Elixir OTP Fault Tolerance & Property Testing ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-elixir
exit 0
