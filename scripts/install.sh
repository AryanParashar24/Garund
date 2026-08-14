#!/bin/sh
set -e

REPO="${GARUND_REPO:-AryanParashar24/Garund}"
BRANCH="${GARUND_BRANCH:-master}"
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

# 4. Resolve Release URL, Local Binary, or Source Fallback
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

  echo "Attempting to download ${BINARY_NAME}..."
  if command -v curl >/dev/null 2>&1; then
    if curl -fsSL "$RELEASE_URL" -o "${TMP_DIR}/garund" 2>/dev/null; then
      DOWNLOAD_SUCCESS=1
    fi
  elif command -v wget >/dev/null 2>&1; then
    if wget -qO "${TMP_DIR}/garund" "$RELEASE_URL" 2>/dev/null; then
      DOWNLOAD_SUCCESS=1
    fi
  fi

  # Source build fallback if release download returns 404/fails
  if [ "$DOWNLOAD_SUCCESS" -ne 1 ]; then
    echo "Notice: Release asset '${BINARY_NAME}' not found on GitHub Releases."
    if command -v go >/dev/null 2>&1; then
      echo "Go toolchain detected ($(go version | awk '{print $3}')). Building Garund from repository source..."
      
      SRC_DIR="${TMP_DIR}/src"
      mkdir -p "$SRC_DIR"
      
      TARBALL_URL="https://github.com/${REPO}/archive/refs/heads/${BRANCH}.tar.gz"
      echo "Fetching repository archive (${BRANCH})..."
      
      BUILD_OK=0
      if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$TARBALL_URL" | tar -xz -C "$SRC_DIR" --strip-components=1 2>/dev/null && BUILD_OK=1
      elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$TARBALL_URL" | tar -xz -C "$SRC_DIR" --strip-components=1 2>/dev/null && BUILD_OK=1
      fi

      if [ "$BUILD_OK" -eq 1 ] && [ -d "$SRC_DIR" ]; then
        echo "Compiling Garund binary..."
        (
          cd "$SRC_DIR"
          go build -o "${TMP_DIR}/garund" ./main.go
        )
        if [ -f "${TMP_DIR}/garund" ]; then
          DOWNLOAD_SUCCESS=1
          echo "✓ Successfully compiled Garund from source."
        fi
      fi
    fi
  fi
fi

if [ "$DOWNLOAD_SUCCESS" -ne 1 ]; then
  echo ""
  echo "Error: Could not install Garund."
  echo ""
  echo "Why this happened:"
  echo "  1. Pre-compiled release assets for '${BINARY_NAME}' are not yet published on GitHub Releases."
  echo "  2. Go development toolchain ('go') was not found on this system to perform automatic compilation."
  echo ""
  echo "To resolve:"
  echo "  Build & install manually from source:"
  echo "      git clone https://github.com/${REPO}.git"
  echo "      cd Garund"
  echo "      make build"
  echo "      make install"
  echo "      export PATH=\"\$HOME/.local/bin:\$PATH\""
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
