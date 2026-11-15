#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Telegram Mini App Go Backend Setup & Build Script for Ubuntu 24.04 LTS
# ==============================================================================

REQUIRED_GO_VERSION="1.23.4"
TARGET_DIR="${HOME}/go_sdk"

echo "==> Step 1: Checking system architecture and prerequisites..."
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        GO_ARCH="amd64"
        ;;
    aarch64|arm64)
        GO_ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Check if Go is already installed
INSTALL_GO=false
if command -v go >/dev/null 2>&1; then
    CURRENT_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "Found existing Go version: $CURRENT_VERSION"
    # Basic check if current version is acceptable
    if [[ "$CURRENT_VERSION" < "1.23" ]]; then
        echo "Go version is below 1.23. Upgrading to $REQUIRED_GO_VERSION..."
        INSTALL_GO=true
    else
        echo "Current Go version is sufficient."
    fi
else
    echo "Go is not installed."
    INSTALL_GO=true
fi

# Install Go if missing or outdated
if [ "$INSTALL_GO" = true ]; then
    echo "==> Step 2: Downloading and setting up Go ${REQUIRED_GO_VERSION} for Linux (${GO_ARCH})..."
    TAR_FILE="go${REQUIRED_GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    DOWNLOAD_URL="https://go.dev/dl/${TAR_FILE}"

    mkdir -p "${TARGET_DIR}"
    curl -fsSL "${DOWNLOAD_URL}" -o "/tmp/${TAR_FILE}"
    rm -rf "${TARGET_DIR}/go"
    tar -C "${TARGET_DIR}" -xzf "/tmp/${TAR_FILE}"
    rm -f "/tmp/${TAR_FILE}"

    export GOROOT="${TARGET_DIR}/go"
    export PATH="${GOROOT}/bin:${PATH}"

    # Persist in user bashrc if not already present
    if ! grep -q "GOROOT=" "${HOME}/.bashrc" 2>/dev/null; then
        echo "export GOROOT=\"${TARGET_DIR}/go\"" >> "${HOME}/.bashrc"
        echo "export PATH=\"\${GOROOT}/bin:\${PATH}\"" >> "${HOME}/.bashrc"
    fi
    echo "Go installed successfully: $(go version)"
fi

# Ensure C compiler and standard build essentials are present (needed for Pebble Cgo/optimizations)
if ! command -v gcc >/dev/null 2>&1; then
    echo "==> Installing build-essential (gcc, make)..."
    if [ "$(id -u)" -eq 0 ]; then
        apt-get update -y && apt-get install -y build-essential curl git
    else
        sudo apt-get update -y && sudo apt-get install -y build-essential curl git
    fi
fi

# Navigate to backend directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

echo "==> Step 3: Downloading and verifying project dependencies..."
go mod download
go mod verify

echo "==> Step 4: Compiling backend binary..."
mkdir -p "${SCRIPT_DIR}/bin"
CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o "${SCRIPT_DIR}/bin/server" ./cmd/server

echo "=============================================================================="
echo "Build succeeded! Executable located at: ${SCRIPT_DIR}/bin/server"
echo "To run server:"
echo "  PORT=8080 ${SCRIPT_DIR}/bin/server"
echo "=============================================================================="
