param(
    [string]$SetupPath = (Join-Path $PSScriptRoot "..\..\dist\Wiz4rd_Fr0g_Setup.exe")
)

$ErrorActionPreference = "Stop"

function Assert-True([bool]$Condition, [string]$Message) {
    if (-not $Condition) { throw $Message }
}

function Wait-Until([scriptblock]$Condition, [int]$Seconds, [string]$Failure) {
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        if (& $Condition) { return }
        Start-Sleep -Milliseconds 250
    }
    throw $Failure
}

function Run-Setup([string[]]$Arguments) {
    $p = Start-Process -FilePath $SetupPath -ArgumentList $Arguments -PassThru -Wait
    if ($p.ExitCode -ne 0) {
        throw "Setup failed: args=$($Arguments -join ' ') exit=$($p.ExitCode)"
    }
}

if ($env:OS -ne "Windows_NT") { throw "Windows is required" }
$SetupPath = (Resolve-Path -LiteralPath $SetupPath).Path

$installDir = Join-Path $env:ProgramFiles "Wiz4rd Fr0g"
$appPath = Join-Path $installDir "Wiz4rdFr0g.exe"
$uninstaller = Join-Path $installDir "Uninstall.exe"
$regPath = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Wiz4rdFr0g"
$shortcut = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Wiz4rd Fr0g.lnk"
$configDir = Join-Path $env:APPDATA "Wiz4rdFr0g"
$settingsPath = Join-Path $configDir "settings.json"

# Start from a controlled state. An external Setup can invoke the idempotent uninstall path.
Run-Setup @("--uninstall", "--quiet")
Wait-Until { -not (Test-Path -LiteralPath $installDir) } 30 "pre-test cleanup did not remove install directory"

# 1) Clean unattended install.
Run-Setup @("--quiet")
Assert-True (Test-Path -LiteralPath $appPath) "installed application missing"
Assert-True (Test-Path -LiteralPath $uninstaller) "self-uninstaller missing"
Assert-True (Test-Path -LiteralPath $shortcut) "Start menu shortcut missing"
Assert-True (Test-Path -LiteralPath $regPath) "Installed Apps registry key missing"

$reg = Get-ItemProperty -LiteralPath $regPath
Assert-True (-not [string]::IsNullOrWhiteSpace($reg.UninstallString)) "UninstallString missing"
Assert-True (-not [string]::IsNullOrWhiteSpace($reg.QuietUninstallString)) "QuietUninstallString missing"
Assert-True ($reg.UninstallString -eq ('"' + $uninstaller + '" --uninstall')) "UninstallString is not exact"
Assert-True ($reg.QuietUninstallString -eq ('"' + $uninstaller + '" --uninstall --quiet')) "QuietUninstallString is not exact"

# 2) Upgrade in place and preserve personal settings.
New-Item -ItemType Directory -Force $configDir | Out-Null
'{"phase2_marker":true}' | Set-Content -Encoding UTF8 -LiteralPath $settingsPath
Run-Setup @("--quiet")
Assert-True (Test-Path -LiteralPath $settingsPath) "upgrade removed personal settings"
Assert-True ((Get-Content -Raw -LiteralPath $settingsPath) -match 'phase2_marker') "upgrade changed personal settings unexpectedly"
Assert-True (Test-Path -LiteralPath $uninstaller) "upgrade lost uninstaller"

# 3) Same-name foreign executable safety. Compile a harmless sleeper with the
# exact Wiz4rdFr0g.exe filename outside Program Files, then ensure upgrade does not kill it.
$foreignDir = Join-Path $env:TEMP ("wf-foreign-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force $foreignDir | Out-Null
$foreignExe = Join-Path $foreignDir "Wiz4rdFr0g.exe"
$src = @'
using System;
using System.Threading;
public static class Program { public static void Main() { Thread.Sleep(60000); } }
'@
Add-Type -TypeDefinition $src -Language CSharp -OutputAssembly $foreignExe -OutputType ConsoleApplication
$foreign = Start-Process -FilePath $foreignExe -PassThru
try {
    Start-Sleep -Milliseconds 500
    Assert-True (-not $foreign.HasExited) "foreign same-name fixture exited before upgrade"
    Run-Setup @("--quiet")
    $foreign.Refresh()
    Assert-True (-not $foreign.HasExited) "upgrade killed a same-name executable outside the install directory"
}
finally {
    if (-not $foreign.HasExited) { Stop-Process -Id $foreign.Id -Force }
    Remove-Item -LiteralPath $foreignDir -Recurse -Force -ErrorAction SilentlyContinue
}

# 4) Quiet uninstall and disappearance verification.
$p = Start-Process -FilePath $uninstaller -ArgumentList @("--uninstall", "--quiet") -PassThru -Wait
if ($p.ExitCode -ne 0) { throw "quiet uninstaller failed with exit $($p.ExitCode)" }
Wait-Until { -not (Test-Path -LiteralPath $appPath) } 30 "application executable still exists after uninstall"
Wait-Until { -not (Test-Path -LiteralPath $installDir) } 30 "install directory still exists after uninstall cleanup"
Assert-True (-not (Test-Path -LiteralPath $regPath)) "uninstall registry key still exists"
Assert-True (-not (Test-Path -LiteralPath $shortcut)) "Start menu shortcut still exists"
Assert-True (Test-Path -LiteralPath $settingsPath) "uninstall removed personal settings"

# 5) Idempotent second uninstall from the external Setup.
Run-Setup @("--uninstall", "--quiet")
Assert-True (-not (Test-Path -LiteralPath $regPath)) "second uninstall recreated/left registry state"

# 6) Partial-install cleanup: no app executable, only an orphan owned file.
New-Item -ItemType Directory -Force $installDir | Out-Null
"orphan" | Set-Content -LiteralPath (Join-Path $installDir "orphan.tmp")
Run-Setup @("--uninstall", "--quiet")
Wait-Until { -not (Test-Path -LiteralPath $installDir) } 30 "partial-install cleanup left the install directory"

Write-Host "SELF_LIFECYCLE_PASS"
