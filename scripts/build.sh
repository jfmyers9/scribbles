#!/bin/bash

# Build script for scribbles
# Supports cross-compilation for macOS (darwin) on both Intel and Apple Silicon

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get version from git tag, or use "dev" if not on a tag
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build output directory
BUILD_DIR="./dist"
BINARY_NAME="scribbles"

# Go build flags
LDFLAGS="-X github.com/jfmyers9/scribbles/cmd.version=${VERSION} \
         -X github.com/jfmyers9/scribbles/cmd.commit=${COMMIT} \
         -X github.com/jfmyers9/scribbles/cmd.buildDate=${BUILD_DATE}"

echo -e "${GREEN}Building scribbles${NC}"
echo "Version:    ${VERSION}"
echo "Commit:     ${COMMIT}"
echo "Build Date: ${BUILD_DATE}"
echo ""

# Clean previous builds
if [ -d "$BUILD_DIR" ]; then
    echo -e "${YELLOW}Cleaning previous builds...${NC}"
    rm -rf "$BUILD_DIR"
fi
mkdir -p "$BUILD_DIR"

# Build for macOS (Intel)
echo -e "${GREEN}Building for darwin/amd64...${NC}"
GOOS=darwin GOARCH=amd64 go build \
    -ldflags "$LDFLAGS" \
    -o "${BUILD_DIR}/${BINARY_NAME}-darwin-amd64" \
    .

# Build for macOS (Apple Silicon)
echo -e "${GREEN}Building for darwin/arm64...${NC}"
GOOS=darwin GOARCH=arm64 go build \
    -ldflags "$LDFLAGS" \
    -o "${BUILD_DIR}/${BINARY_NAME}-darwin-arm64" \
    .

# Create universal binary (optional, for convenience)
echo -e "${GREEN}Creating universal binary...${NC}"
lipo -create \
    "${BUILD_DIR}/${BINARY_NAME}-darwin-amd64" \
    "${BUILD_DIR}/${BINARY_NAME}-darwin-arm64" \
    -output "${BUILD_DIR}/${BINARY_NAME}"

# Generate checksums
echo -e "${GREEN}Generating checksums...${NC}"
cd "$BUILD_DIR"
shasum -a 256 ${BINARY_NAME}-darwin-amd64 > ${BINARY_NAME}-darwin-amd64.sha256
shasum -a 256 ${BINARY_NAME}-darwin-arm64 > ${BINARY_NAME}-darwin-arm64.sha256
shasum -a 256 ${BINARY_NAME} > ${BINARY_NAME}.sha256
cd - > /dev/null

# Create archives
echo -e "${GREEN}Creating release archives...${NC}"
cd "$BUILD_DIR"
tar -czf "${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz" ${BINARY_NAME}-darwin-amd64
tar -czf "${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz" ${BINARY_NAME}-darwin-arm64
tar -czf "${BINARY_NAME}-${VERSION}-darwin-universal.tar.gz" ${BINARY_NAME}
cd - > /dev/null

echo -e "${GREEN}✓ Build complete!${NC}"
echo ""
echo "Artifacts:"
ls -lh "${BUILD_DIR}"

echo ""
echo -e "${YELLOW}To test the binary:${NC}"
echo "  ${BUILD_DIR}/${BINARY_NAME} --version"
echo ""
echo -e "${YELLOW}To install locally:${NC}"
echo "  cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/"
