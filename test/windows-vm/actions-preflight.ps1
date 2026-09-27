param([Parameter(Mandatory=$true)][int]$Index)
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
$proof = [ordered]@{
    status = 'BLOCKED_ENVIRONMENT'
    generated_at = [DateTime]::UtcNow.ToString('o')
    vm_id = $env:WIZ4RDFR0G_VM_ID
    test_run_id = $env:WIZ4RDFR0G_TEST_RUN_ID
    git_commit = $env:GITHUB_SHA
    runner_environment = $env:RUNNER_ENVIRONMENT
    temp_directory = $env:TEMP
    tmp_directory = $env:TMP
    architecture = $env:RUNNER_ARCH
    windows_version = ''
    image_os = $env:ImageOS
    image_version = $env:ImageVersion
    isolation = 'one catalog index per fresh GitHub-hosted job'
    commands = @()
}
try {
    if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted' -or $env:RUNNER_OS -ne 'Windows' -or $env:RUNNER_ARCH -ne 'X64') { throw 'Approved GitHub-hosted Windows x64 runner required' }
    $run = "github-$env:GITHUB_RUN_ID-$env:GITHUB_RUN_ATTEMPT"
    if ($env:WIZ4RDFR0G_VM_ID -ne "$run-$Index" -or $env:WIZ4RDFR0G_TEST_RUN_ID -ne $run) { throw 'Run/VM identity mismatch' }
    if (-not $env:WIZ4RDFR0G_EVIDENCE_HMAC_KEY -or $env:WIZ4RDFR0G_EVIDENCE_HMAC_KEY.Length -lt 32) { throw 'Repository evidence HMAC secret missing or too short' }
    $osInfo = Get-CimInstance Win32_OperatingSystem
    $proof.windows_version = "$($osInfo.Caption) $($osInfo.Version) $($osInfo.OSArchitecture)"
    $proof.last_boot = $osInfo.LastBootUpTime.ToUniversalTime().ToString('o')
    $goVersion = (& go version | Out-String).Trim()
    $proof.commands += @{command='go version'; exit_code=$LASTEXITCODE; output=$goVersion}
    if ($LASTEXITCODE -ne 0 -or $goVersion -notmatch 'go(\d+\.\d+(?:\.\d+)?)') { throw 'Go version unavailable' }
    $actualGo = [version]$Matches[1]
    $minimum = [regex]::Match((Get-Content go.mod -Raw), '(?m)^go\s+(\S+)').Groups[1].Value
    if ($actualGo -lt [version]$minimum) { throw 'Go version below go.mod requirement' }
    $network = Invoke-WebRequest -Uri 'https://github.com/microsoft/winget-pkgs' -Method Head -TimeoutSec 30
    $proof.internet = @{url='https://github.com/microsoft/winget-pkgs'; http_status=[int]$network.StatusCode}
    $wingetReady = $false
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        $initialVersion = (& winget --version 2>&1 | Out-String)
        $proof.commands += @{command='winget --version (initial)'; exit_code=$LASTEXITCODE; output=$initialVersion}
        $wingetReady = $LASTEXITCODE -eq 0
    }
    if (-not $wingetReady) {
        # Microsoft-supported bootstrap, only after hosted-VM authorization checks.
        Install-Module -Name Microsoft.WinGet.Client -Repository PSGallery -Scope CurrentUser -Force
        Import-Module Microsoft.WinGet.Client
        $repair = Repair-WinGetPackageManager -AllUsers | Out-String
        $proof.bootstrap = @{method='Microsoft.WinGet.Client / Repair-WinGetPackageManager -AllUsers'; output=$repair; module_version=(Get-Module Microsoft.WinGet.Client).Version.ToString()}
        $package = Get-AppxPackage -Name Microsoft.DesktopAppInstaller | Select-Object -First 1
        if ($package -and (Test-Path -LiteralPath (Join-Path $package.InstallLocation 'winget.exe'))) {
            $env:PATH = "$($package.InstallLocation);$env:PATH"
            $package.InstallLocation | Add-Content -LiteralPath $env:GITHUB_PATH
        }
    }
    foreach ($arguments in @(@('--version'), @('source','list'), @('source','update','--name','winget'), @('show','--id','Microsoft.PowerToys','--exact','--source','winget','--accept-source-agreements','--disable-interactivity'))) {
        $ready = $false
        for($attempt=1; $attempt -le 3; $attempt++) {
            $output = (& winget @arguments 2>&1 | Out-String)
            $code = $LASTEXITCODE
            $proof.commands += @{command=@('winget')+$arguments; attempt=$attempt; exit_code=$code; output=$output}
            $ready = $code -eq 0 -and $output -notmatch '(?im)^\s*Cancelled\s*$'
            if($ready){break}
            if($attempt -lt 3){
                if($output -match '0x8a15000f|Data required by the source is missing') {
                    $repair = (& winget source update --name winget 2>&1 | Out-String)
                    $proof.commands += @{command=@('winget','source','update','--name','winget'); attempt=$attempt; exit_code=$LASTEXITCODE; output=$repair}
                }
                Start-Sleep -Seconds 2
            }
        }
        if (-not $ready) { throw "Winget preflight failed after bounded retries: $arguments" }
    }
    $proof.status = 'READY'
} catch {
    $proof.reason = $_.Exception.Message
} finally {
    $parent = Split-Path -Parent $env:WIZ4RDFR0G_ENVIRONMENT_EVIDENCE
    New-Item -ItemType Directory -Force -Path $parent | Out-Null
    $proof | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $env:WIZ4RDFR0G_ENVIRONMENT_EVIDENCE -Encoding utf8NoBOM
}
if ($proof.status -ne 'READY') { throw "BLOCKED_ENVIRONMENT: $($proof.reason)" }
