#!/usr/bin/env bash
set -euo pipefail

# Outputs the next date-based version: YYYY.M.PATCH
TODAY="$(date +%Y.%-m)"
LATEST="$(git tag --list "${TODAY}.*" 2>/dev/null | sort -V | tail -1)"

if [ -z "$LATEST" ]; then
  echo "${TODAY}.0"
else
  PATCH="${LATEST##*.}"
  echo "${TODAY}.$((PATCH + 1))"
fi
