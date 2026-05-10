#!/usr/bin/env bash
# install.sh — myhelper installer
# Usage: curl -sfL https://raw.githubusercontent.com/bkohler93/myhelper/main/install.sh | bash
#
# Environment overrides:
#   INSTALL_DIR   Override install directory (default: ~/.local/bin)
#
# Examples:
#   curl -sfL https://raw.githubusercontent.com/bkohler93/myhelper/main/install.sh | bash
#   curl -sfL https://raw.githubusercontent.com/bkohler93/myhelper/main/install.sh | INSTALL_DIR=/usr/local/bin bash
set -euo pipefail

REPO="bkohler93/myhelper"
BINARY="myhelper"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

# ── OS / arch detection ────────────────────────────────────────────────────────
uname_os() {
  local os
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    mingw*) os="windows" ;;
    cygwin*) os="linux" ;;
  esac
  echo "$os"
}

uname_arch() {
  local arch
  arch=$(uname -m)
  case "$arch" in
    x86_64)  arch="amd64" ;;
    aarch64) arch="arm64" ;;
    armv7*)  arch="arm" ;;
  esac
  echo "$arch"
}

OS=$(uname_os)
ARCH=$(uname_arch)

# ── Fetch latest release tag ───────────────────────────────────────────────────
TAG=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/')

if [[ -z "$TAG" ]]; then
  echo "Error: could not determine latest release tag from GitHub API." >&2
  echo "  Check https://github.com/${REPO}/releases for the latest version." >&2
  exit 1
fi

VERSION="${TAG#v}"   # strip leading 'v' — archive names use "1.0.0" not "v1.0.0"

# ── Construct download URLs ────────────────────────────────────────────────────
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUM="${BINARY}_${VERSION}_checksums.txt"

echo "Installing ${BINARY} ${TAG} (${OS}/${ARCH})..."

# ── Download to temp dir ───────────────────────────────────────────────────────
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -sfL "${BASE_URL}/${ARCHIVE}" -o "${TMP}/${ARCHIVE}"
curl -sfL "${BASE_URL}/${CHECKSUM}" -o "${TMP}/${CHECKSUM}"

# ── Verify checksum ────────────────────────────────────────────────────────────
cd "$TMP"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum --ignore-missing -c "${CHECKSUM}"
elif command -v shasum >/dev/null 2>&1; then
  # macOS ships shasum, not sha256sum
  grep "${ARCHIVE}" "${CHECKSUM}" | shasum -a 256 -c -
else
  echo "Error: sha256sum and shasum not found — cannot verify checksum. Aborting." >&2
  exit 1
fi
cd - >/dev/null

# ── Extract and install ────────────────────────────────────────────────────────
tar -xzf "${TMP}/${ARCHIVE}" -C "${TMP}"
mkdir -p "${INSTALL_DIR}"
cp "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

echo "Installed ${BINARY} ${TAG} to ${INSTALL_DIR}/${BINARY}"

# ── PATH hint ─────────────────────────────────────────────────────────────────
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
  echo ""
  echo "NOTE: ${INSTALL_DIR} is not in your PATH."
  echo "Add this to your ~/.bashrc or ~/.zshrc:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi
