#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
OUT="dist"
MODULE="github.com/t0mer/aghsync"
LDFLAGS="-s -w -X ${MODULE}/cmd/aghsync.version=${VERSION}"

# Build frontend first if web/package.json exists
if [ -f "web/package.json" ]; then
  echo "Building frontend..."
  (cd web && npm ci --silent && npm run build)
  echo "Copying web/dist → internal/webui/dist/"
  rm -rf internal/webui/dist
  cp -r web/dist internal/webui/dist
fi

mkdir -p "$OUT"

declare -A targets=(
  ["linux-amd64"]="linux/amd64/"
  ["linux-arm64"]="linux/arm64/"
  ["linux-armv7"]="linux/arm/7"
  ["linux-armv6"]="linux/arm/6"
  ["linux-386"]="linux/386/"
  ["darwin-amd64"]="darwin/amd64/"
  ["darwin-arm64"]="darwin/arm64/"
  ["windows-amd64"]="windows/amd64/"
  ["windows-arm64"]="windows/arm64/"
)

for name in "${!targets[@]}"; do
  IFS='/' read -r GOOS GOARCH GOARM <<< "${targets[$name]}"
  ext=""
  [ "$GOOS" = "windows" ] && ext=".exe"
  outfile="${OUT}/aghsync-${name}${ext}"
  echo "Building ${outfile}..."
  env GOOS="$GOOS" GOARCH="$GOARCH" GOARM="${GOARM:-}" \
    go build -ldflags "$LDFLAGS" -o "$outfile" ./cmd/aghsync/
done

echo "Done. Artifacts in ${OUT}/"
