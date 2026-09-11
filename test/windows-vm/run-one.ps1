param(
    [Parameter(Mandatory=$true)][int]$Index,
    [switch]$Resume,
    [string]$Exe = ".\dist\Wiz4rdFr0g.exe",
    [string]$ResultsDir = ".\test\windows-vm\results"
)
$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path $ResultsDir | Out-Null
$result = Join-Path $ResultsDir ("{0:D4}.json" -f $Index)
$env:WIZ4RDFR0G_VM_TEST = "1"
if (-not $env:WIZ4RDFR0G_PACKAGE_RETRIES) { $env:WIZ4RDFR0G_PACKAGE_RETRIES = "1" }
if (-not $env:WIZ4RDFR0G_UNINSTALL_RETRIES) { $env:WIZ4RDFR0G_UNINSTALL_RETRIES = "1" }
if (-not $env:WIZ4RDFR0G_VM_ID) { $env:WIZ4RDFR0G_VM_ID = "local-$env:COMPUTERNAME-$Index" }
if (-not $env:WIZ4RDFR0G_TEST_RUN_ID) { $env:WIZ4RDFR0G_TEST_RUN_ID = "local-$([guid]::NewGuid().ToString("N"))" }
if (-not $env:WIZ4RDFR0G_EVIDENCE_HMAC_KEY -or $env:WIZ4RDFR0G_EVIDENCE_HMAC_KEY.Length -lt 32) { throw "Set WIZ4RDFR0G_EVIDENCE_HMAC_KEY to a secret with at least 32 characters before physical testing." }
$mode = "--vm-test-one"
if ($Resume) { $mode = "--vm-test-resume" }
& $Exe $mode $Index $result
$code = $LASTEXITCODE
if (-not (Test-Path $result)) { throw "VM test did not produce a result file: $result" }
Get-Content $result
exit $code
