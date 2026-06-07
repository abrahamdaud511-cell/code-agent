#!/bin/bash
# CodeAgent Installer
# Usage: curl -fsSL https://codeagent.ai/install.sh | sh

set -e

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "Installing CodeAgent ${VERSION}..."

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${ARCH}" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

BINARY_URL="https://github.com/anomalyco/codeagent/releases/download/${VERSION}/codeagent-${OS}-${ARCH}"

if command -v curl &>/dev/null; then
    curl -fsSL "${BINARY_URL}" -o "${INSTALL_DIR}/codeagent"
elif command -v wget &>/dev/null; then
    wget -qO "${INSTALL_DIR}/codeagent" "${BINARY_URL}"
else
    echo "Need curl or wget to install"
    exit 1
fi

chmod +x "${INSTALL_DIR}/codeagent"
echo "CodeAgent installed to ${INSTALL_DIR}/codeagent"
echo "Run 'codeagent init' to get started"
