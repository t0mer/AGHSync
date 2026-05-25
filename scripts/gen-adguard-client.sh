#!/usr/bin/env bash
set -euo pipefail
# Requires: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

mkdir -p internal/adguard/gen
oapi-codegen --config=scripts/oapi-codegen-config.yaml swagger.yml
echo "Client regenerated at internal/adguard/gen/adguard.gen.go"
