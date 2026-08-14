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
if [ -n "$GARUND_INSTALL_DIR" ]; then
  INSTALL_DIR="$GARUND_INSTALL_DIR"
elif [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
fi

mkdir -p "$INSTALL_DIR"

TARGET_BIN="${INSTALL_DIR}/garund"
if [ "$OS" = "windows" ]; then
  TARGET_BIN="${TARGET_BIN}.exe"
fi

echo "✓ Target installation path: ${TARGET_BIN}"

# 4. Resolve Release URL or Local Binary
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

DOWNLOAD_SUCCESS=0

if [ -f "bin/garund" ]; then
  echo "Found local compiled binary at bin/garund. Installing..."
  cp bin/garund "${TMP_DIR}/garund"
  DOWNLOAD_SUCCESS=1
else
  if [ "$VERSION" = "latest" ]; then
    RELEASE_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"
  else
    RELEASE_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
  fi

  echo "Downloading ${BINARY_NAME} from ${RELEASE_URL}..."
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
  echo "         make install"
  echo ""
  exit 1
fi

chmod +x "${TMP_DIR}/garund"
mv "${TMP_DIR}/garund" "$TARGET_BIN"

echo "✓ Successfully installed Garund to ${TARGET_BIN}"

# 5. PATH check & export instructions
case ":$PATH:" in
  *":${INSTALL_DIR}:"*)
    echo "✓ ${INSTALL_DIR} is in your PATH."
    ;;
  *)
    echo ""
    echo "Notice: ${INSTALL_DIR} is not in your current PATH environment variable."
    echo ""
    echo "To start using garund right away in your current shell session, export it:"
    echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    echo "To persist this change for future shell sessions, add it to your profile:"
    echo "    echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc   # for bash"
    echo "    echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.zshrc    # for zsh"
    echo ""
    ;;
esac

echo ""
echo "Garund installation complete!"
echo "Verify version:"
echo "    garund version"
echo ""
echo "Start Garund:"
echo "    garund start"
