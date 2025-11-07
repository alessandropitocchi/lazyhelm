#!/bin/bash
set -e

# LazyHelm Installation Script

REPO="alessandropitocchi/lazyhelm"
BINARY_NAME="lazyhelm"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing LazyHelm...${NC}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo "Detected: $OS/$ARCH"

# Get latest release
echo "Fetching latest release..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${YELLOW}No releases found. Installing from source...${NC}"

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Go is not installed. Please install Go 1.21+ first.${NC}"
        exit 1
    fi

    # Clone and build
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    echo "Cloning repository..."
    git clone "https://github.com/$REPO.git"
    cd lazyhelm
    echo "Building..."
    go build -o "$BINARY_NAME" ./cmd/lazyhelm

    # Install binary
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
    else
        echo "Installing to $INSTALL_DIR (requires sudo)..."
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
    fi

    cd ~
    rm -rf "$TEMP_DIR"
else
    echo "Latest release: $LATEST_RELEASE"

    # Download binary
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/${BINARY_NAME}_${OS}_${ARCH}"
    TEMP_FILE=$(mktemp)

    echo "Downloading..."
    if curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
        chmod +x "$TEMP_FILE"

        # Install binary
        if [ -w "$INSTALL_DIR" ]; then
            mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
        else
            echo "Installing to $INSTALL_DIR (requires sudo)..."
            sudo mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
        fi
    else
        echo -e "${RED}Failed to download binary. Trying from source...${NC}"
        rm "$TEMP_FILE"

        # Fallback to source installation
        if ! command -v go &> /dev/null; then
            echo -e "${RED}Go is not installed. Please install Go 1.21+ first.${NC}"
            exit 1
        fi

        go install "github.com/$REPO/cmd/lazyhelm@latest"
        echo -e "${GREEN}Installed via 'go install'${NC}"
        exit 0
    fi
fi

# Verify installation
if command -v $BINARY_NAME &> /dev/null; then
    echo -e "${GREEN}✓ LazyHelm installed successfully!${NC}"
    echo ""
    echo "Run 'lazyhelm' to get started"
    echo ""
    echo "Optional: Set your preferred editor"
    echo "  export EDITOR=nvim"
else
    echo -e "${RED}Installation failed${NC}"
    exit 1
fi
