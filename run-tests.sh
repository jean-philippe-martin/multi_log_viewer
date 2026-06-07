#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "==> go vet"
go vet ./...

echo "==> go test"
go test ./... -count=1 "$@"

echo "==> golangci-lint"
if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run ./...
else
  go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
fi

echo "OK"
