#!/usr/bin/env bash
set -euo pipefail
# Requires: go install github.com/air-verse/air@latest  and  Node 20+

trap 'kill 0' EXIT

echo "Starting frontend dev server..."
(cd web && npm run dev) &

echo "Starting Go backend with hot reload..."
air -- --port 8080 --log-level debug &

wait
