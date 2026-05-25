#!/usr/bin/env bash
set -euo pipefail

VERSION="${SWAGGER_UI_VERSION:-5.18.2}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="$REPO_ROOT/internal/api/docs/swagger-ui"

mkdir -p "$DEST"

echo "Downloading swagger-ui-dist v${VERSION}..."
curl -fsSL "https://unpkg.com/swagger-ui-dist@${VERSION}/swagger-ui-bundle.js" \
     -o "$DEST/swagger-ui-bundle.js"
curl -fsSL "https://unpkg.com/swagger-ui-dist@${VERSION}/swagger-ui.css" \
     -o "$DEST/swagger-ui.css"

# Verify integrity of vendored assets for known versions
if [ "$VERSION" = "5.18.2" ]; then
  echo "Verifying SHA256 integrity for swagger-ui v${VERSION}..."
  echo "c50b94bbc4f02394326fb7aed1f4fb693b3677f4b3d3344e0d6131808cbf281f  $DEST/swagger-ui-bundle.js" | sha256sum -c
  echo "8f33d996025317049d4a9864f421eab2b2a247872f388026fa94c654913259e7  $DEST/swagger-ui.css" | sha256sum -c
  echo "Integrity check passed."
fi

echo "Vendored swagger-ui v${VERSION} to $DEST/"
