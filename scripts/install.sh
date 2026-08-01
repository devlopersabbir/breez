#!/usr/bin/env bash
set -e

# Breez installer script for Linux & macOS

REPO="devlopersabbir/breez"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="breez"

echo "Downloading latest release of Breez..."

# Detect OS & Arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

LATEST_RELEASE=$(curl -s https://api.github.com/repos/${REPO}/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo "Error: Unable to fetch latest release tag for $REPO."
    exit 1
fi

TARBALL_NAME="breez_${LATEST_RELEASE#v}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_RELEASE}/${TARBALL_NAME}"

TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "Downloading ${DOWNLOAD_URL}..."
curl -sL "$DOWNLOAD_URL" -o "$TEMP_DIR/$TARBALL_NAME"

tar -xzf "$TEMP_DIR/$TARBALL_NAME" -C "$TEMP_DIR"

if [ ! -w "$INSTALL_DIR" ]; then
    echo "Installing binary to $INSTALL_DIR (requires sudo)..."
    sudo mv "$TEMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    echo "Installing binary to $INSTALL_DIR..."
    mv "$TEMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

echo "✔ Breez successfully installed to $INSTALL_DIR/$BINARY_NAME"
echo "Run 'breez version' or 'breez serve <port>' to get started!"
