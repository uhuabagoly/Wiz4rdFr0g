param(
    [Parameter(Mandatory=$true)][int]$Start,
    [Parameter(Mandatory=$true)][int]$Count,
    [string]$Exe = ".\dist\Wiz4rdFr0g.exe",
    [string]$ResultsDir = ".\test\windows-vm\results",
    [string]$Manifest = ".\release\build_manifest.json"
)
$ErrorActionPreference = "Stop"
if ($Count -lt 1 -or $Count -gt 25) { throw "Count must be between 1 and 25." }
if (-not $env:WIZ4RDFR0G_EVIDENCE_HMAC_KEY -or $env:WIZ4RDFR0G_EVIDENCE_HMAC_KEY.Length -lt 32) { throw "Set WIZ4RDFR0G_EVIDENCE_HMAC_KEY to a secret with at least 32 characters before physical testing." }
if (-not (Test-Path -LiteralPath $Manifest)) { throw "Missing build manifest: $Manifest" }
New-Item -ItemType Directory -Force -Path $ResultsDir | Out-Null
for ($i = $Start; $i -lt ($Start + $Count); $i++) {
    $result = Join-Path $ResultsDir ("{0:D4}.json" -f $i)
    if (Test-Path -LiteralPath $result) {
        & go run .\cmd\evidence-check $result $Manifest
        $validCode = $LASTEXITCODE
        if ($validCode -eq 0) {
            $existing = Get-Content $result -Raw | ConvertFrom-Json
            $terminal = @("FULL_PASS", "SYSTEM_COMPONENT", "MANUAL_ONLY", "LICENSE_REQUIRED")
            if ($terminal -contains [string]$existing.final_status) {
                Write-Host "SKIP authenticated current-build result $i -> $($existing.final_status)"
                continue
            }
            Write-Host "RETEST authenticated non-terminal result $i -> $($existing.final_status)"
        } else {
            Write-Warning "Existing result $i is stale/invalid for this build; it will be replaced."
        }
    }
    $env:WIZ4RDFR0G_VM_TEST = "1"
    $env:WIZ4RDFR0G_VM_ID = "local-$env:COMPUTERNAME-$i"
    $env:WIZ4RDFR0G_TEST_RUN_ID = "campaign-$([guid]::NewGuid().ToString('N'))"
    & $Exe --vm-test-one $i $result
    $code = $LASTEXITCODE
    if ($code -ne 0) {
        Write-Warning "Catalog index $i did not pass. The result contains diagnosis/repair attempts: $result"
    }
}
go run .\cmd\vm-test-report $ResultsDir
