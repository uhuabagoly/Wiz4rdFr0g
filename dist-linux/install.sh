#!/usr/bin/env bash
set -euo pipefail

APP_NAME="Wiz4rd Fr0g"
BIN_NAME="Wiz4rdFr0g"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "Ez a telepítő Linuxhoz készült."
  exit 1
fi

if ! command -v sha256sum >/dev/null 2>&1; then
  echo "A csomag integritásának ellenőrzéséhez sha256sum szükséges."
  exit 3
fi
if [[ ! -f "$SCRIPT_DIR/SHA256SUMS" ]]; then
  echo "Hiányzik a SHA256SUMS fájl; a telepítés biztonsági okból megszakadt."
  exit 4
fi
if ! (cd "$SCRIPT_DIR" && sha256sum -c SHA256SUMS); then
  echo "A kiadási csomag integritás-ellenőrzése sikertelen."
  exit 5
fi

if ! ldconfig -p 2>/dev/null | grep -F 'libgtk-3.so.0' >/dev/null; then
  echo "A Wiz4rd Fr0g Linux GUI-jához GTK3 szükséges (libgtk-3.so.0)."
  echo "A program nem telepít automatikusan extra runtime-ot."
  exit 2
fi

INSTALL_DIR="$HOME/.local/opt/wiz4rd-fr0g"
APP_DIR="$HOME/.local/share/applications"
ICON_DIR="$HOME/.local/share/icons/hicolor/256x256/apps"
mkdir -p "$INSTALL_DIR" "$APP_DIR" "$ICON_DIR"

install -m 0755 "$SCRIPT_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
install -m 0644 "$SCRIPT_DIR/Wiz4rdFr0g_icon.png" "$INSTALL_DIR/Wiz4rdFr0g_icon.png"
install -m 0644 "$SCRIPT_DIR/Wiz4rdFr0g_icon.png" "$ICON_DIR/wiz4rd-fr0g.png"

cat > "$APP_DIR/wiz4rd-fr0g.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Wiz4rd Fr0g
Comment=Ingyenes programok telepítőközpontja
Exec=$INSTALL_DIR/$BIN_NAME
Icon=wiz4rd-fr0g
Terminal=false
Categories=Utility;System;
StartupNotify=true
DESKTOP
chmod 0644 "$APP_DIR/wiz4rd-fr0g.desktop"

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$APP_DIR" >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" >/dev/null 2>&1 || true
fi

if [[ ! -x "$INSTALL_DIR/$BIN_NAME" || ! -f "$APP_DIR/wiz4rd-fr0g.desktop" || ! -f "$ICON_DIR/wiz4rd-fr0g.png" ]]; then
  echo "A telepítés utóellenőrzése sikertelen."
  exit 6
fi

echo "$APP_NAME telepítve."
echo "Indítás: $INSTALL_DIR/$BIN_NAME"
echo "Az alkalmazásmenüben is meg kell jelennie: $APP_NAME"
