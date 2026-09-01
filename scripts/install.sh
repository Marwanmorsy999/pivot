#!/usr/bin/env bash
set -euo pipefail

# PIVOT Install Script
# Downloads and installs the latest release binary

REPO="Marwanmorsy999/pivot"
INSTALL_DIR="${PIVOT_INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="pivot"

echo "📦 Installing PIVOT..."

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    msys*|mingw*|cygwin*) OS="windows" ;;
    *) echo "❌ Unsupported OS: $OS"; exit 1 ;;
esac

# Determine binary name
if [[ "$OS" == "windows" ]]; then
    BINARY_NAME="pivot.exe"
    SUFFIX=".exe"
else
    SUFFIX=""
fi

BINARY_FILE="pivot-${OS}-${ARCH}${SUFFIX}"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Get latest release version
echo "🔍 Fetching latest release..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | cut -d'"' -f4)
if [[ -z "$LATEST_RELEASE" ]]; then
    echo "❌ Failed to fetch latest release"
    exit 1
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_RELEASE}/${BINARY_FILE}"

echo "⬇️  Downloading ${BINARY_FILE} (${LATEST_RELEASE})..."
curl -sL "$DOWNLOAD_URL" -o "${INSTALL_DIR}/${BINARY_NAME}"

chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo ""
echo "✅ PIVOT installed successfully!"
echo "   Version: ${LATEST_RELEASE}"
echo "   Location: ${INSTALL_DIR}/${BINARY_NAME}"
echo ""

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo "⚠️  Warning: ${INSTALL_DIR} is not in your PATH"
    echo "   Add this line to your shell config (.bashrc, .zshrc, etc.):"
    echo "   export PATH=\"${INSTALL_DIR}:\$PATH\""
fi

echo ""
echo "Run 'pivot --help' to get started!"
