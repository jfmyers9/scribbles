#!/bin/bash
#
# Installation script for scribbles
# Usage: curl -fsSL https://raw.githubusercontent.com/jfmyers9/scribbles/main/scripts/install.sh | bash
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="jfmyers9/scribbles"
BINARY_NAME="scribbles"
INSTALL_DIR="/usr/local/bin"

echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║          scribbles Installation Script                ║${NC}"
echo -e "${BLUE}║     Apple Music Scrobbler for Last.fm                 ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
echo ""

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        PLATFORM="darwin-amd64"
        ;;
    arm64)
        PLATFORM="darwin-arm64"
        ;;
    *)
        echo -e "${RED}✗ Unsupported architecture: $ARCH${NC}"
        echo "  scribbles only supports macOS on Intel (x86_64) or Apple Silicon (arm64)"
        exit 1
        ;;
esac

echo -e "${GREEN}✓${NC} Detected platform: macOS ($ARCH)"

# Get latest release version
echo -e "${YELLOW}→${NC} Fetching latest release..."
LATEST_VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}✗ Failed to fetch latest version${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Latest version: $LATEST_VERSION"

# Download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}-${LATEST_VERSION}-${PLATFORM}.tar.gz"
CHECKSUM_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}-${PLATFORM}.sha256"

# Create temporary directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

echo -e "${YELLOW}→${NC} Downloading scribbles $LATEST_VERSION for $PLATFORM..."
cd "$TMP_DIR"

if ! curl -fsSL -o "${BINARY_NAME}.tar.gz" "$DOWNLOAD_URL"; then
    echo -e "${RED}✗ Failed to download scribbles${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Downloaded successfully"

# Verify checksum (optional but recommended)
if curl -fsSL -o "${BINARY_NAME}.sha256" "$CHECKSUM_URL" 2>/dev/null; then
    echo -e "${YELLOW}→${NC} Verifying checksum..."
    # Extract just the hash from the checksum file
    EXPECTED_HASH=$(cat "${BINARY_NAME}.sha256" | awk '{print $1}')
    ACTUAL_HASH=$(shasum -a 256 "${BINARY_NAME}.tar.gz" | awk '{print $1}')

    if [ "$EXPECTED_HASH" = "$ACTUAL_HASH" ]; then
        echo -e "${GREEN}✓${NC} Checksum verified"
    else
        echo -e "${RED}✗ Checksum verification failed${NC}"
        echo "  Expected: $EXPECTED_HASH"
        echo "  Got:      $ACTUAL_HASH"
        exit 1
    fi
fi

# Extract archive
echo -e "${YELLOW}→${NC} Extracting archive..."
tar -xzf "${BINARY_NAME}.tar.gz"

# Install binary
echo -e "${YELLOW}→${NC} Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    cp "${BINARY_NAME}-${PLATFORM}" "$INSTALL_DIR/$BINARY_NAME"
else
    echo -e "${YELLOW}  (requires sudo)${NC}"
    sudo cp "${BINARY_NAME}-${PLATFORM}" "$INSTALL_DIR/$BINARY_NAME"
fi

# Make executable
if [ -w "$INSTALL_DIR/$BINARY_NAME" ]; then
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

echo -e "${GREEN}✓${NC} scribbles installed successfully"
echo ""

# Verify installation
INSTALLED_VERSION=$($BINARY_NAME --version 2>/dev/null | head -1)
echo -e "${GREEN}✓${NC} Verification: $INSTALLED_VERSION"
echo ""

# Next steps
echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    Next Steps                          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "1. ${YELLOW}Authenticate with Last.fm:${NC}"
echo -e "   ${GREEN}scribbles auth${NC}"
echo ""
echo -e "2. ${YELLOW}Install the background daemon:${NC}"
echo -e "   ${GREEN}scribbles install${NC}"
echo ""
echo -e "3. ${YELLOW}Test the CLI (optional):${NC}"
echo -e "   ${GREEN}scribbles now${NC}"
echo ""
echo -e "For more information, visit:"
echo -e "  ${BLUE}https://github.com/$REPO${NC}"
echo ""
