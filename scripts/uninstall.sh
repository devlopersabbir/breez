#!/usr/bin/env bash
set -e

# Breez uninstaller script for Linux & macOS

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="breez"
TARGET_PATH="$INSTALL_DIR/$BINARY_NAME"

if [ ! -f "$TARGET_PATH" ]; then
    echo "Breez binary not found at $TARGET_PATH."
    exit 0
fi

if [ ! -w "$INSTALL_DIR" ]; then
    echo "Removing $TARGET_PATH (requires sudo)..."
    sudo rm -f "$TARGET_PATH"
else
    echo "Removing $TARGET_PATH..."
    rm -f "$TARGET_PATH"
fi

CONFIG_DIR="$HOME/.config/breez"
if [ -d "$CONFIG_DIR" ]; then
    echo "Removing local configuration at $CONFIG_DIR..."
    rm -rf "$CONFIG_DIR"
fi

echo "✔ Breez has been successfully uninstalled from your system."
