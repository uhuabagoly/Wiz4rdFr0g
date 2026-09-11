$ErrorActionPreference = "Stop"

go run ./cmd/release-build ./release/build_manifest.json
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Get-Content ./dist-linux/SHA256SUMS | ForEach-Object {
    if ($_ -notmatch '^([0-9a-fA-F]{64})  (.+)$') {
        throw "Invalid SHA256SUMS entry: $_"
    }
    $expected = $Matches[1].ToLowerInvariant()
    $path = Join-Path ./dist-linux $Matches[2]
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Missing Linux release payload: $path"
    }
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $path).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "SHA256 mismatch for $path"
    }
}

go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go vet ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go vet ./app ./installer
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue

go run ./cmd/catalog-audit -out ./audit
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
New-Item -ItemType Directory -Force ./test/windows-vm/results | Out-Null
New-Item -ItemType Directory -Force ./release | Out-Null
go run ./cmd/vm-test-report ./test/windows-vm/results
$gate = Join-Path $env:TEMP "wiz4rdfr0g-release-gate.exe"
go build -o $gate ./cmd/release-gate
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& $gate ./test/windows-vm/results ./release ./release/build_manifest.json
exit $LASTEXITCODE
