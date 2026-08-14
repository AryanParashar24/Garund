#!/bin/sh
set -e

PURGE=0

for arg in "$@"; do
  case "$arg" in
    --purge) PURGE=1 ;;
  esac
done

echo "Garund Uninstaller"
echo "─────────────────────────────────────────"

INSTALL_LOCATIONS=""

if [ -n "$GARUND_INSTALL_DIR" ]; then
  INSTALL_LOCATIONS="$GARUND_INSTALL_DIR/garund $GARUND_INSTALL_DIR/garund.exe"
else
  INSTALL_LOCATIONS="$HOME/.local/bin/garund $HOME/.local/bin/garund.exe /usr/local/bin/garund /usr/local/bin/garund.exe"
fi

REMOVED=0

for loc in $INSTALL_LOCATIONS; do
  if [ -f "$loc" ]; then
    echo "Removing binary at $loc..."
    rm -f "$loc"
    REMOVED=$((REMOVED + 1))
    echo "✓ Removed $loc"
  fi
done

if [ "$REMOVED" -eq 0 ]; then
  echo "Notice: No Garund binaries found in standard installation directories."
fi

if [ "$PURGE" -eq 1 ]; then
  DATA_DIR="$HOME/.garund"
  if [ -d "$DATA_DIR" ]; then
    echo "Purging configuration and runtime directory at $DATA_DIR..."
    rm -rf "$DATA_DIR"
    echo "✓ Purged $DATA_DIR"
  fi
else
  echo ""
  echo "Note: User configuration and runtime data in ~/.garund/ were preserved."
  echo "To remove configuration and data as well, run:"
  echo "    ./scripts/uninstall.sh --purge"
fi

echo ""
echo "Garund uninstallation complete."
