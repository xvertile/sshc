#!/bin/bash

INSTALL_DIR="/usr/local/bin"
EXECUTABLE_NAME=sshc
EXECUTABLE_PATH="$INSTALL_DIR/$EXECUTABLE_NAME"
USE_SUDO="false"
OS=""
ARCH=""
FORCE_INSTALL="${FORCE_INSTALL:-false}"
SSHC_VERSION="${SSHC_VERSION:-latest}"

RED='\033[0;31m'
PURPLE='\033[0;35m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

usage() {
    printf "${PURPLE}SSHC Installation Script${NC}\n\n"
    printf "Usage:\n"
    printf "  Default (latest stable):     ${GREEN}bash install.sh${NC}\n"
    printf "  Specific version:            ${GREEN}SSHC_VERSION=v1.0.0 bash install.sh${NC}\n"
    printf "  Force install:               ${GREEN}FORCE_INSTALL=true bash install.sh${NC}\n"
    printf "  Custom install directory:    ${GREEN}INSTALL_DIR=/opt/bin bash install.sh${NC}\n\n"
    printf "Environment variables:\n"
    printf "  SSHC_VERSION    - Version to install (default: latest)\n"
    printf "  FORCE_INSTALL   - Skip confirmation prompts (default: false)\n"
    printf "  INSTALL_DIR     - Installation directory (default: /usr/local/bin)\n\n"
}

setSystem() {
    ARCH=$(uname -m)
    case $ARCH in
        i386|i686) ARCH="amd64" ;;
        x86_64) ARCH="amd64";;
        aarch64*) ARCH="arm64" ;;
        arm64) ARCH="arm64" ;;
    esac

    OS=$(echo `uname`|tr '[:upper:]' '[:lower:]')

    if [ "$OS" = "linux" ] || [ "$OS" = "darwin" ]; then
        USE_SUDO="true"
    fi
}

runAsRoot() {
    local CMD="$*"
    if [ "$USE_SUDO" = "true" ]; then
        printf "${PURPLE}We need sudo access to install SSHC to $INSTALL_DIR ${NC}\n"
        CMD="sudo $CMD"
    fi
    $CMD
}

getLatestVersion() {
    if [ "$SSHC_VERSION" = "latest" ]; then
        printf "${YELLOW}Fetching latest stable version...${NC}\n"
        LATEST_VERSION=$(curl -s https://api.github.com/repos/xvertile/sshc/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$LATEST_VERSION" ]; then
            printf "${RED}Failed to fetch latest version${NC}\n"
            exit 1
        fi
    else
        printf "${YELLOW}Using specified version: $SSHC_VERSION${NC}\n"
        RELEASE_CHECK=$(curl -s "https://api.github.com/repos/xvertile/sshc/releases/tags/$SSHC_VERSION" | grep '"tag_name":')
        if [ -z "$RELEASE_CHECK" ]; then
            printf "${RED}Version $SSHC_VERSION not found. Available versions:${NC}\n"
            curl -s https://api.github.com/repos/xvertile/sshc/releases | grep '"tag_name":' | head -10 | sed -E 's/.*"([^"]+)".*/  - \1/'
            exit 1
        fi
        LATEST_VERSION="$SSHC_VERSION"
    fi
    printf "${GREEN}Installing version: $LATEST_VERSION${NC}\n"
}

downloadBinary() {
    # GoReleaser format: sshc_v1.0.0_linux_amd64.tar.gz
    GITHUB_FILE="sshc_${LATEST_VERSION}_${OS}_${ARCH}.tar.gz"
    GITHUB_URL="https://github.com/xvertile/sshc/releases/download/$LATEST_VERSION/$GITHUB_FILE"

    printf "${YELLOW}Downloading $GITHUB_FILE...${NC}\n"
    curl -L "$GITHUB_URL" --progress-bar --output "sshc-tmp.tar.gz"

    if [ $? -ne 0 ]; then
        printf "${RED}Failed to download binary${NC}\n"
        exit 1
    fi

    tar -xzf "sshc-tmp.tar.gz"
    if [ $? -ne 0 ]; then
        printf "${RED}Failed to extract binary${NC}\n"
        exit 1
    fi

    EXTRACTED_BINARY="./sshc"
    if [ ! -f "$EXTRACTED_BINARY" ]; then
        printf "${RED}Could not find extracted binary: $EXTRACTED_BINARY${NC}\n"
        exit 1
    fi

    mv "$EXTRACTED_BINARY" "sshc-tmp"
    rm -f "sshc-tmp.tar.gz"
}

install() {
    printf "${YELLOW}Installing SSHC...${NC}\n"

    OLD_BACKUP=""
    if [ -f "$EXECUTABLE_PATH" ]; then
        OLD_BACKUP="$EXECUTABLE_PATH.backup.$$"
        runAsRoot mv "$EXECUTABLE_PATH" "$OLD_BACKUP"
    fi

    chmod +x "sshc-tmp"
    if [ $? -ne 0 ]; then
        printf "${RED}Failed to set permissions${NC}\n"
        if [ -n "$OLD_BACKUP" ] && [ -f "$OLD_BACKUP" ]; then
            runAsRoot mv "$OLD_BACKUP" "$EXECUTABLE_PATH"
        fi
        exit 1
    fi

    runAsRoot mv "sshc-tmp" "$EXECUTABLE_PATH"
    if [ $? -ne 0 ]; then
        printf "${RED}Failed to install binary${NC}\n"
        if [ -n "$OLD_BACKUP" ] && [ -f "$OLD_BACKUP" ]; then
            runAsRoot mv "$OLD_BACKUP" "$EXECUTABLE_PATH"
        fi
        exit 1
    fi

    if [ -n "$OLD_BACKUP" ] && [ -f "$OLD_BACKUP" ]; then
        runAsRoot rm -f "$OLD_BACKUP"
    fi
}

cleanup() {
    rm -f "sshc-tmp" "sshc-tmp.tar.gz" "sshc" LICENSE README.md
}

checkExisting() {
    if command -v sshc >/dev/null 2>&1; then
        CURRENT_VERSION=$(sshc --version 2>/dev/null | grep -o 'version.*' | cut -d' ' -f2 || echo "unknown")
        printf "${YELLOW}SSHC is already installed (version: $CURRENT_VERSION)${NC}\n"

        if [ "$FORCE_INSTALL" = "true" ]; then
            printf "${GREEN}Force install enabled, proceeding with installation...${NC}\n"
            return
        fi

        if [ ! -t 0 ]; then
            printf "${YELLOW}Running via pipe - automatically proceeding with installation...${NC}\n"
            return
        fi

        printf "${YELLOW}Do you want to overwrite it? [y/N]: ${NC}"
        read -r response
        case "$response" in
            [yY][eE][sS]|[yY])
                printf "${GREEN}Proceeding with installation...${NC}\n"
                ;;
            *)
                printf "${GREEN}Installation cancelled.${NC}\n"
                exit 0
                ;;
        esac
    fi
}

main() {
    if [ "$1" = "-h" ] || [ "$1" = "--help" ] || [ "$1" = "help" ]; then
        usage
        exit 0
    fi

    printf "${PURPLE}Installing SSHC - SSH Client${NC}\n\n"

    setSystem
    printf "${GREEN}Detected system: $OS ($ARCH)${NC}\n"

    getLatestVersion
    checkExisting
    downloadBinary
    install
    cleanup

    printf "\n${GREEN}SSHC was installed successfully to: ${NC}$EXECUTABLE_PATH\n"
    printf "${GREEN}You can now use 'sshc' command!${NC}\n\n"

    printf "${YELLOW}Verifying installation...${NC}\n"
    if command -v sshc >/dev/null 2>&1; then
        "$EXECUTABLE_PATH" --version 2>/dev/null || echo "Version check failed, but installation completed"
    else
        printf "${RED}Warning: 'sshc' command not found in PATH. You may need to restart your terminal.${NC}\n"
    fi
}

trap cleanup EXIT

main "$@"
