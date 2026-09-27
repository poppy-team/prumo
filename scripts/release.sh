#!/bin/sh
set -eu

VERSION="${VERSION:-0.6.0}"
OUTPUT="${OUTPUT:-dist}"

mkdir -p "$OUTPUT"

targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"

for target in $targets; do
  os="${target%%/*}"
  arch="${target##*/}"
  ext=""
  if [ "$os" = "windows" ]; then
    ext=".exe"
  fi

  # 1. Prumo Harness CLI
  cli_name="prumo-harness-cli-${os}-${arch}${ext}"
  legacy_name="prumo-${os}-${arch}${ext}"
  echo "building $cli_name"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUTPUT/$cli_name" ./cmd/prumo
  cp "$OUTPUT/$cli_name" "$OUTPUT/$legacy_name"

  # 2. Prumo Agent TUI
  tui_name="prumo-agent-tui-${os}-${arch}${ext}"
  echo "building $tui_name"
  (cd prumo-tui && GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "../$OUTPUT/$tui_name" ./cmd/prumo-tui)
done

# 3. Prumo Agent IDE (Desktop GUI)
if command -v cargo >/dev/null 2>&1; then
  host_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  host_arch="$(uname -m)"
  case "$host_arch" in
    x86_64) host_arch="amd64" ;;
    aarch64) host_arch="arm64" ;;
  esac
  ext=""
  if [ "$host_os" = "windows" ] || [ "${OS:-}" = "Windows_NT" ]; then
    ext=".exe"
  fi
  ide_name="prumo-agent-ide-${host_os}-${host_arch}${ext}"
  echo "packaging $ide_name"
  if [ ! -f "prumo-viewer/target/release/prumo-native${ext}" ]; then
    (cd prumo-viewer && cargo build --release --bin prumo-native)
  fi
  cp "prumo-viewer/target/release/prumo-native${ext}" "$OUTPUT/$ide_name"
fi

(cd "$OUTPUT" && sha256sum prumo-* > checksums.txt)

artifacts_json=$(cd "$OUTPUT" && ls -1 prumo-* | awk '{printf "    \"%s\",\n", $0}' | sed '$ s/,$//')

cat > "$OUTPUT/release.json" <<EOF
{
  "version": "$VERSION",
  "artifacts": [
$artifacts_json
  ]
}
EOF

echo "release artifacts written to $OUTPUT"

