#!/usr/bin/env bash
set -euo pipefail

VERSION="${SWAGGER_UI_VERSION:-5.18.2}"
DEST="internal/api/docs/swagger-ui"

mkdir -p "$DEST"

echo "Downloading swagger-ui-dist v${VERSION}..."
curl -fsSL "https://unpkg.com/swagger-ui-dist@${VERSION}/swagger-ui-bundle.js" \
     -o "$DEST/swagger-ui-bundle.js"
curl -fsSL "https://unpkg.com/swagger-ui-dist@${VERSION}/swagger-ui.css" \
     -o "$DEST/swagger-ui.css"

echo "Vendored swagger-ui v${VERSION} to $DEST/"
