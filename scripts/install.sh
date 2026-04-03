#!/bin/sh
# Celador CLI Universal Installer
# Usage: curl -fsSL https://codexlighthouse.com/celador/install.sh | sh

set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
VERSION=${CELADOR_VERSION:-latest}

echo "Downloading Celador CLI binary metadata for ${OS}/${ARCH} (${VERSION})..."
echo "This repository now expects a precompiled Go binary distribution."
echo "Until release assets are published, build locally with:"
echo "  go build -o celador ./cmd/celador"
echo "Then move the binary into your PATH, for example:"
echo "  install -m 0755 ./celador /usr/local/bin/celador"
