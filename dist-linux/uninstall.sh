#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="$HOME/.local/opt/wiz4rd-fr0g"
APP_DIR="$HOME/.local/share/applications"
ICON_ROOT="$HOME/.local/share/icons/hicolor"
ICON_FILE="$ICON_ROOT/256x256/apps/wiz4rd-fr0g.png"
DESKTOP_FILE="$APP_DIR/wiz4rd-fr0g.desktop"

# Only Wiz4rd Fr0g-owned installation artifacts are removed. User settings are
# intentionally preserved.
rm -rf -- "$INSTALL_DIR"
rm -f -- "$DESKTOP_FILE"
rm -f -- "$ICON_FILE"

if command -v update-desktop-database >/dev/null 2>&1 && [[ -d "$APP_DIR" ]]; then
  update-desktop-database "$APP_DIR" >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1 && [[ -d "$ICON_ROOT" ]]; then
  gtk-update-icon-cache -f -t "$ICON_ROOT" >/dev/null 2>&1 || true
fi

if [[ -e "$INSTALL_DIR" || -e "$DESKTOP_FILE" || -e "$ICON_FILE" ]]; then
  echo "A Wiz4rd Fr0g eltávolításának utóellenőrzése sikertelen." >&2
  exit 2
fi

echo "Wiz4rd Fr0g eltávolítva a felhasználói profilból. A személyes beállítások megmaradtak."
