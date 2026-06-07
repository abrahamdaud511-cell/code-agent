#!/usr/bin/env bash
set -euo pipefail

# CodeAgent Build Script
# Builds for all platforms

PROJECT="codeagent"
VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
OUTPUT_DIR="./dist"

PLATFORMS=(
	"linux/amd64"
	"linux/arm64"
	"darwin/amd64"
	"darwin/arm64"
	"windows/amd64"
)

echo "Building $PROJECT v$VERSION"
echo ""

mkdir -p "$OUTPUT_DIR"

for platform in "${PLATFORMS[@]}"; do
	OS="${platform%/*}"
	ARCH="${platform#*/}"

	output_name="$PROJECT-${OS}-${ARCH}"
	if [ "$OS" = "windows" ]; then
		output_name="${output_name}.exe"
	fi

	echo "Building for $OS/$ARCH..."
	GOOS="$OS" GOARCH="$ARCH" CGO_ENABLED=0 go build \
		-ldflags="-s -w -X github.com/anomalyco/codeagent/cmd.Version=$VERSION" \
		-o "$OUTPUT_DIR/$output_name" \
		.

	echo "  -> $OUTPUT_DIR/$output_name"
done

echo ""
echo "Build complete! Binaries in $OUTPUT_DIR:"
ls -lh "$OUTPUT_DIR"
