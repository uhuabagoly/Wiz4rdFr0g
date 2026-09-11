param(
    [Parameter(Mandatory=$true)][int]$Start,
    [Parameter(Mandatory=$true)][int]$Count,
    [string]$Exe = ".\dist\Wiz4rdFr0g.exe",
    [string]$ResultsDir = ".\test\windows-vm\results"
)
$ErrorActionPreference = "Stop"
if ($Count -lt 1 -or $Count -gt 25) { throw "Count must be between 1 and 25." }
New-Item -ItemType Directory -Force -Path $ResultsDir | Out-Null
for ($i = $Start; $i -lt ($Start + $Count); $i++) {
    $result = Join-Path $ResultsDir ("{0:D4}.json" -f $i)
    if (Test-Path $result) {
        $existing = Get-Content $result -Raw | ConvertFrom-Json
        if ($existing.final_status) {
            Write-Host "SKIP existing result $i -> $($existing.final_status)"
            continue
        }
    }
    $env:WIZ4RDFR0G_VM_TEST = "1"
    $env:WIZ4RDFR0G_VM_ID = "local-$env:COMPUTERNAME-$i"
    & $Exe --vm-test-one $i $result
    $code = $LASTEXITCODE
    if ($code -ne 0) { Write-Warning "Catalog index $i failed. Evidence: $result" }
}
go run .\cmd\vm-test-report $ResultsDir
