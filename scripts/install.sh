#!/bin/sh
set -eu

REPOSITORY="${PRUMO_REPOSITORY:-raillen/prumo}"
VERSION="${PRUMO_VERSION:-v0.6.1}"
INSTALL_DIR="${PRUMO_INSTALL_DIR:-${HOME}/.local/bin}"
PRUMO_HOME_VALUE="${PRUMO_HOME:-${HOME}/.prumo}"
PACKAGE="${PRUMO_PACKAGE:-${1:-}}"
BASE_URL="https://github.com/${REPOSITORY}/releases/download/${VERSION}"

case "$(uname -s)" in
  Linux) OS="linux" ;;
  Darwin) OS="darwin" ;;
  *) printf '%s\n' "Unsupported operating system. Use a release asset or build from source." >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) printf '%s\n' "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [ -z "$PACKAGE" ]; then
  if [ -t 0 ]; then
    printf '%s\n' "================================================="
    printf '%s\n' "       Prumo Package Selection ($VERSION)"
    printf '%s\n' "================================================="
    printf '%s\n' "1) Prumo CLI        (Core runtime, daemon & tools) [default]"
    printf '%s\n' "2) Prumo IDE        (Desktop GUI pair programmer & editor)"
    printf '%s\n' "3) Prumo Code Agent (Interactive terminal client)"
    printf '%s\n' "4) All packages      (CLI + IDE + Code Agent)"
    printf '%s' "Select package [1-4, default: 1]: "
    read -r choice || choice=""
    case "$choice" in
      2|ide|IDE) PACKAGE="ide" ;;
      3|tui|TUI) PACKAGE="tui" ;;
      4|all|ALL) PACKAGE="all" ;;
      *) PACKAGE="cli" ;;
    esac
  else
    PACKAGE="cli"
  fi
fi

case "$PACKAGE" in
  cli|harness)
    INSTALL_CLI=true
    INSTALL_IDE=false
    INSTALL_TUI=false
    ;;
  ide|viewer)
    INSTALL_CLI=false
    INSTALL_IDE=true
    INSTALL_TUI=false
    ;;
  tui)
    INSTALL_CLI=false
    INSTALL_IDE=false
    INSTALL_TUI=true
    ;;
  all|full)
    INSTALL_CLI=true
    INSTALL_IDE=true
    INSTALL_TUI=true
    ;;
  *)
    printf '%s\n' "Unknown package: $PACKAGE. Choose from: cli, ide, tui, all." >&2
    exit 1
    ;;
esac

if ! command -v curl >/dev/null 2>&1; then
  printf '%s\n' "curl is required." >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  CHECKSUM_TOOL="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  CHECKSUM_TOOL="shasum -a 256"
else
  printf '%s\n' "sha256sum or shasum is required." >&2
  exit 1
fi

TEMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT INT TERM

curl --fail --silent --show-error --location "${BASE_URL}/checksums.txt" -o "${TEMP_DIR}/checksums.txt"

install_artifact() {
  asset="$1"
  dest_name="$2"

  expected="$(awk -v a="$asset" '$2 == a {print $1}' "${TEMP_DIR}/checksums.txt")"
  if [ -z "$expected" ]; then
    printf '%s\n' "No checksum found for ${asset}." >&2
    return 1
  fi

  printf '%s\n' "Downloading ${asset}..."
  curl --fail --silent --show-error --location "${BASE_URL}/${asset}" -o "${TEMP_DIR}/${asset}"

  if [ "$CHECKSUM_TOOL" = "sha256sum" ]; then
    actual="$(sha256sum "${TEMP_DIR}/${asset}" | awk '{print $1}')"
  else
    actual="$(shasum -a 256 "${TEMP_DIR}/${asset}" | awk '{print $1}')"
  fi

  if [ "$expected" != "$actual" ]; then
    printf '%s\n' "Checksum verification failed for ${asset}." >&2
    return 1
  fi

  mkdir -p "$INSTALL_DIR"
  chmod 0755 "${TEMP_DIR}/${asset}"
  target="${INSTALL_DIR}/${dest_name}"
  staged="${target}.tmp.$$"
  cp "${TEMP_DIR}/${asset}" "$staged"
  chmod 0755 "$staged"
  mv "$staged" "$target"
  printf '%s\n' "Installed ${dest_name} at ${target}."
}

if [ "$INSTALL_CLI" = "true" ]; then
  cli_asset="prumo-harness-cli-${OS}-${ARCH}"
  if ! grep -q " ${cli_asset}\$" "${TEMP_DIR}/checksums.txt" 2>/dev/null; then
    cli_asset="prumo-${OS}-${ARCH}"
  fi
  install_artifact "$cli_asset" "prumo"
  PRUMO_HOME="$PRUMO_HOME_VALUE" "${INSTALL_DIR}/prumo" setup >/dev/null || true
fi

if [ "$INSTALL_TUI" = "true" ]; then
  tui_asset="prumo-agent-tui-${OS}-${ARCH}"
  install_artifact "$tui_asset" "prumo-tui"
  ln -sf "${INSTALL_DIR}/prumo-tui" "${INSTALL_DIR}/pa" || cp "${INSTALL_DIR}/prumo-tui" "${INSTALL_DIR}/pa"
fi

if [ "$INSTALL_IDE" = "true" ]; then
  ide_asset="prumo-agent-ide-${OS}-${ARCH}"
  install_artifact "$ide_asset" "prumo-ide"
  ln -sf "${INSTALL_DIR}/prumo-ide" "${INSTALL_DIR}/prumo-viewer" || cp "${INSTALL_DIR}/prumo-ide" "${INSTALL_DIR}/prumo-viewer"
fi

# Configure system PATH idempotently
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    # Already in PATH
    ;;
  *)
    UPDATED_FILES=""
    if [ -f "${HOME}/.bashrc" ] && ! grep -qF "$INSTALL_DIR" "${HOME}/.bashrc"; then
      printf '\n# Added by Prumo\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "${HOME}/.bashrc"
      UPDATED_FILES="${UPDATED_FILES} ~/.bashrc"
    fi

    if [ -f "${HOME}/.zshrc" ] && ! grep -qF "$INSTALL_DIR" "${HOME}/.zshrc"; then
      printf '\n# Added by Prumo\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "${HOME}/.zshrc"
      UPDATED_FILES="${UPDATED_FILES} ~/.zshrc"
    fi

    if [ -f "${HOME}/.config/fish/config.fish" ] && ! grep -qF "$INSTALL_DIR" "${HOME}/.config/fish/config.fish"; then
      printf '\n# Added by Prumo\nfish_add_path "%s"\n' "$INSTALL_DIR" >> "${HOME}/.config/fish/config.fish"
      UPDATED_FILES="${UPDATED_FILES} ~/.config/fish/config.fish"
    fi

    if [ -f "${HOME}/.profile" ] && ! grep -qF "$INSTALL_DIR" "${HOME}/.profile"; then
      printf '\n# Added by Prumo\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "${HOME}/.profile"
      UPDATED_FILES="${UPDATED_FILES} ~/.profile"
    fi

    if [ -n "$UPDATED_FILES" ]; then
      printf '%s\n' "Configured PATH in:${UPDATED_FILES}"
      printf '%s\n' "To start using prumo in this terminal session, run:"
      printf '%s\n' "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
    ;;
esac

printf '%s\n' ""
printf '%s\n' "Prumo installation complete! Quick start:"
if [ "$INSTALL_CLI" = "true" ]; then
  printf '%s\n' "  prumo version       - Check CLI version"
  printf '%s\n' "  prumo serve         - Start background daemon"
fi
if [ "$INSTALL_TUI" = "true" ]; then
  printf '%s\n' "  prumo-tui (or pa)   - Launch interactive terminal harness"
fi
if [ "$INSTALL_IDE" = "true" ]; then
  printf '%s\n' "  prumo-ide           - Launch native desktop Agent IDE"
fi
