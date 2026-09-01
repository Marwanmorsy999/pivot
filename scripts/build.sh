#!/usr/bin/env bash
set -euo pipefail

# PIVOT Build Script
# Builds binaries for multiple platforms

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
LDFLAGS="-ldflags \"-s -w -X main.version=${VERSION}\"\""

echo "🔨 Building PIVOT v${VERSION}..."

# Create dist directory
mkdir -p dist

# Build for different platforms
platforms=(
    "linux/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    if [[ "$GOOS" == "windows" ]]; then
        OUTPUT="dist/pivot-${GOOS}-${GOARCH}.exe"
    else
        OUTPUT="dist/pivot-${GOOS}-${GOARCH}"
    fi
    
    echo "   Building for ${GOOS}/${GOARCH}..."
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build $LDFLAGS -o "$OUTPUT" ./cmd/pivot
done

echo "✅ Build complete! Binaries in dist/"
ls -lh dist/
