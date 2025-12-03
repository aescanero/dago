#!/bin/bash

# Build script for DA Orchestrator

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Variables
BINARY_NAME="dago"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-ldflags \"-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}\""

echo -e "${GREEN}Building DA Orchestrator${NC}"
echo "Version: ${VERSION}"
echo "Build Time: ${BUILD_TIME}"
echo ""

# Clean previous builds
echo -e "${YELLOW}Cleaning previous builds...${NC}"
rm -rf bin/ dist/
rm -f ${BINARY_NAME}

# Build for current platform
echo -e "${YELLOW}Building for current platform...${NC}"
CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/dago

echo -e "${GREEN}Build complete: ${BINARY_NAME}${NC}"

# Optionally build for multiple platforms
if [ "$1" == "all" ]; then
    echo -e "${YELLOW}Building for multiple platforms...${NC}"

    mkdir -p dist

    # Linux amd64
    echo "Building for Linux amd64..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 ./cmd/dago

    # Linux arm64
    echo "Building for Linux arm64..."
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-arm64 ./cmd/dago

    # macOS amd64
    echo "Building for macOS amd64..."
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 ./cmd/dago

    # macOS arm64
    echo "Building for macOS arm64..."
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 ./cmd/dago

    # Windows amd64
    echo "Building for Windows amd64..."
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe ./cmd/dago

    echo -e "${GREEN}All builds complete!${NC}"
    echo "Binaries available in dist/"
    ls -lh dist/
fi
