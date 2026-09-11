#!/usr/bin/env bash
set -euo pipefail
go test ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./app ./installer
go run ./cmd/catalog-audit -out ./audit
mkdir -p ./test/windows-vm/results ./release
go run ./cmd/vm-test-report ./test/windows-vm/results || true
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
go build -o "$tmp" ./cmd/release-gate
"$tmp" ./test/windows-vm/results ./release
