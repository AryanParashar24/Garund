#!/bin/sh
set -e

REPO="${GARUND_REPO:-AryanParashar24/Garund}"
VERSION="${GARUND_VERSION:-latest}"

echo "Garund Installer"
echo "─────────────────────────────────────────"

# 1. Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)   OS="linux" ;;
  darwin*)  OS="darwin" ;;
  msys*|mingw*|cygwin*) OS="windows" ;;
  *)
    echo "Error: Unsupported operating system: $OS"
    exit 1
    ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

BINARY_NAME="garund-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  BINARY_NAME="${BINARY_NAME}.exe"
fi

echo "✓ Detected platform: ${OS}/${ARCH}"

# 3. Determine Installation Path
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

TARGET_BIN="${INSTALL_DIR}/garund"
if [ "$OS" = "windows" ]; then
  TARGET_BIN="${TARGET_BIN}.exe"
fi

echo "✓ Target installation path: ${TARGET_BIN}"

# 4. Resolve Release URL
if [ "$VERSION" = "latest" ]; then
  RELEASE_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"
else
  RELEASE_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${BINARY_NAME}..."
DOWNLOAD_SUCCESS=0

if command -v curl >/dev/null 2>&1; then
  if curl -fsSL "$RELEASE_URL" -o "${TMP_DIR}/garund"; then
    DOWNLOAD_SUCCESS=1
  fi
elif command -v wget >/dev/null 2>&1; then
  if wget -qO "${TMP_DIR}/garund" "$RELEASE_URL"; then
    DOWNLOAD_SUCCESS=1
  fi
else
  echo "Error: Neither curl nor wget is available."
  exit 1
fi

if [ "$DOWNLOAD_SUCCESS" -ne 1 ]; then
  echo ""
  echo "Error: Could not download pre-compiled binary '${BINARY_NAME}' from GitHub Releases."
  echo ""
  echo "Why this happened:"
  echo "  No GitHub Release assets exist yet under https://github.com/${REPO}/releases."
  echo ""
  echo "To resolve:"
  echo "  1. Publish a GitHub Release (e.g. tag 'v0.1.0'), OR"
  echo "  2. Build from source locally:"
  echo "         git clone https://github.com/${REPO}.git"
  echo "         cd Garund"
  echo "         make build"
  echo "         cp bin/garund ${TARGET_BIN}"
  echo ""
  exit 1
fi

chmod +x "${TMP_DIR}/garund"
mv "${TMP_DIR}/garund" "$TARGET_BIN"

echo "✓ Successfully installed Garund to ${TARGET_BIN}"

# 5. PATH check
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Notice: ${INSTALL_DIR} is not in your PATH."
    echo "Add it to your profile:"
    echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

echo ""
echo "Garund is ready!"
echo "Run:"
echo "    garund start"
