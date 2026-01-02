#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing sshc...${NC}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

# Get latest release version from GitHub
echo "Fetching latest release..."
LATEST=$(curl -fsSL https://api.github.com/repos/xvertile/sshc/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
VERSION=${LATEST#v}

echo -e "Latest version: ${GREEN}$LATEST${NC}"

# Download URL
if [ "$OS" = "darwin" ]; then
    URL="https://github.com/xvertile/sshc/releases/download/${LATEST}/sshc_${VERSION}_darwin_${ARCH}.tar.gz"
elif [ "$OS" = "linux" ]; then
    URL="https://github.com/xvertile/sshc/releases/download/${LATEST}/sshc_${VERSION}_linux_${ARCH}.tar.gz"
else
    echo -e "${RED}Unsupported OS: $OS${NC}"
    exit 1
fi

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download and extract
echo "Downloading from $URL..."
curl -fsSL "$URL" | tar xz -C "$TMP_DIR"

# Install binary
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Need sudo to install to $INSTALL_DIR${NC}"
    sudo mv "$TMP_DIR/sshc" "$INSTALL_DIR/sshc"
    sudo chmod +x "$INSTALL_DIR/sshc"
else
    mv "$TMP_DIR/sshc" "$INSTALL_DIR/sshc"
    chmod +x "$INSTALL_DIR/sshc"
fi

echo -e "${GREEN}sshc $LATEST installed successfully!${NC}"
echo ""
sshc --version
