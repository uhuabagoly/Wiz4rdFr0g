#!/usr/bin/env bash
set -euo pipefail

go run ./cmd/release-build ./release/build_manifest.json
(cd ./dist-linux && sha256sum -c SHA256SUMS)
go test ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./app ./installer
audit_rc=0
go run ./cmd/catalog-audit -out ./audit || audit_rc=$?
if [[ $audit_rc -ne 0 && $audit_rc -ne 1 ]]; then exit $audit_rc; fi
mkdir -p ./test/windows-vm/results ./release
go run ./cmd/campaign-plan ./test/windows-vm
go run ./cmd/vm-test-report ./test/windows-vm/results || true
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
go build -o "$tmp" ./cmd/release-gate
"$tmp" ./test/windows-vm/results ./release ./release/build_manifest.json
