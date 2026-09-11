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
if (-not $env:WIZ4RDFR0G_VM_ID) { $env:WIZ4RDFR0G_VM_ID = "local-$env:COMPUTERNAME-$Index" }
$mode = "--vm-test-one"
if ($Resume) { $mode = "--vm-test-resume" }
& $Exe $mode $Index $result
$code = $LASTEXITCODE
if (-not (Test-Path $result)) { throw "VM test did not produce a result file: $result" }
Get-Content $result
exit $code
