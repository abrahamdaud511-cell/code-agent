#!/bin/sh
set -eu

REPO="anomalyco/codeagent"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

if [ "${VERSION}" = "latest" ]; then
	VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${ARCH}" in
	x86_64|amd64) ARCH="amd64" ;;
	aarch64|arm64) ARCH="arm64" ;;
	*) echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

case "${OS}" in
	linux|darwin) EXT="tar.gz" ;;
	*) echo "Unsupported OS: ${OS}"; exit 1 ;;
esac

BINARY="codeagent_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"

echo "Downloading CodeAgent ${VERSION} for ${OS}/${ARCH}..."
echo "  ${URL}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

if command -v curl >/dev/null 2>&1; then
	curl -fsSL "${URL}" -o "${TMP_DIR}/${BINARY}"
elif command -v wget >/dev/null 2>&1; then
	wget -qO "${TMP_DIR}/${BINARY}" "${URL}"
else
	echo "Error: need curl or wget"
	exit 1
fi

cd "${TMP_DIR}"
if [ "${EXT}" = "tar.gz" ]; then
	tar xzf "${BINARY}"
else
	unzip "${BINARY}"
fi

mkdir -p "${INSTALL_DIR}"
install codeagent "${INSTALL_DIR}/codeagent"

echo ""
echo "✓ CodeAgent ${VERSION} installed to ${INSTALL_DIR}/codeagent"
echo ""
echo "Quick start:"
echo "  codeagent auth login --provider openai --key sk-..."
echo "  codeagent"
