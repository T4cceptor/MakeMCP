#!/bin/bash
set -e

# MakeMCP Installation Script
# This script downloads and installs the latest version of MakeMCP

REPO="T4cceptor/MakeMCP"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="makemcp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

info() {
    echo -e "${GREEN}$1${NC}"
}

warn() {
    echo -e "${YELLOW}$1${NC}"
}

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case $arch in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
    
    case $os in
        linux|darwin) ;;
        *) error "Unsupported operating system: $os" ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
main() {
    info "Installing MakeMCP..."
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        error "curl is required but not installed"
    fi
    
    # Detect platform
    platform=$(detect_platform)
    info "Detected platform: $platform"
    
    # Get latest version
    info "Fetching latest release..."
    version=$(get_latest_version)
    if [ -z "$version" ]; then
        error "Failed to get latest version"
    fi
    info "Latest version: $version"
    
    # Construct download URL
    filename="makemcp-${version}-${platform}.tar.gz"
    url="https://github.com/${REPO}/releases/download/${version}/${filename}"
    
    # Create temporary directory
    temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT
    
    # Download
    info "Downloading $filename..."
    if ! curl -L -o "$temp_dir/$filename" "$url"; then
        error "Failed to download $url"
    fi
    
    # Extract
    info "Extracting..."
    if ! tar -xzf "$temp_dir/$filename" -C "$temp_dir"; then
        error "Failed to extract $filename"
    fi
    
    # Install
    info "Installing to $INSTALL_DIR..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$temp_dir/$BINARY_NAME" "$INSTALL_DIR/"
    else
        warn "Installing to $INSTALL_DIR requires sudo"
        sudo mv "$temp_dir/$BINARY_NAME" "$INSTALL_DIR/"
    fi
    
    # Make executable
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    # Verify installation
    if "$INSTALL_DIR/$BINARY_NAME" --version &> /dev/null; then
        info "✅ MakeMCP installed successfully!"
        info "Run 'makemcp --help' to get started"
    else
        error "Installation verification failed"
    fi
}

# Run main function
main "$@"