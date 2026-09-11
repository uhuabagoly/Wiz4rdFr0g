# Wiz4rd Fr0g

Wiz4rd Fr0g is a cross-platform program installer and remover for Windows and Linux.

## Platforms

- Windows x64: native Windows application using Winget / Microsoft Store resolution and Windows uninstall metadata.
- Linux x64: native GTK3 application using the package backend available on the active distribution.
- Windows and Linux backends are separated so packages from the other operating system cannot be installed accidentally.

## Catalog

The curated catalog contains 723 entries. Package resolution is exact-only: known package IDs are preferred, exact display-name resolution is used where verified, and ambiguous fuzzy results are rejected.

## Windows install verification

A successful installer exit code is not enough. After installation Wiz4rd Fr0g re-runs the production detector and verifies the installed instance through Winget and/or Windows uninstall metadata before marking the application installed.

## Windows uninstall engine

The uninstall engine resolves the installed instance first, then determines scope, installer type and strategy before executing anything.

Supported strategies include Winget user scope, Winget machine scope, Registry user scope, Registry machine scope, MSI ProductCode, Microsoft Store/MSIX and verified vendor uninstallers. Windows-managed system components and unresolved identities are not passed into unsafe fallback removal.

User-scope uninstall runs in the logged-in user context. Machine-scope uninstall requests UAC only when needed. MSI exit codes 0, 1641 and 3010 are handled with restart semantics. Registered EXE uninstall commands are parsed as executable plus arguments instead of concatenated `cmd /C` strings.

If an uninstaller reports that the application is running, Wiz4rd Fr0g only considers processes whose full executable path is tied to the detected real installation directory or exact DisplayIcon executable. It asks for approval before closing them. No process is terminated from a name substring match.

Microsoft Edge is never confused with WebView2 Runtime, updater or helper components. Edge automatic removal is attempted only when the exact `Microsoft.Edge` package is exposed by Windows as an uninstallable package; otherwise the program does not use Registry/MSI force-removal fallbacks.

## Wiz4rd Fr0g self-install and self-uninstall

The Windows Setup installs a dedicated `Uninstall.exe` and registers both `UninstallString` and `QuietUninstallString` under the standard machine-wide Windows uninstall registry key. Interactive removal asks for confirmation; quiet removal is unattended. The installed application process is stopped only when its kernel-reported executable path exactly matches the Wiz4rd Fr0g installation path, so a same-named executable from another directory is not terminated.

Machine-wide work (`Program Files`, HKLM and self-uninstall cleanup) runs elevated. Start-menu shortcut and per-user installer metadata are created or removed by the original non-elevated process after the machine operation completes. This prevents a standard user's files from being written into a different administrator profile when alternate UAC credentials are supplied. Personal settings are intentionally preserved during self-uninstall.

## Linux install and removal

The Linux GUI supports both installation and removal for resolved native-package and Flatpak entries. Destructive operations use the exact resolved package identifier only; no fuzzy name-based package removal is used. Before removal the package is independently detected, and after either install or removal the package state is checked again before the operation is reported successful.

Supported native removal backends are APT, DNF/RPM, Pacman and Zypper, plus exact user-scope Flatpak application IDs. If a package identity cannot be resolved safely, the operation is rejected instead of falling back to a fuzzy uninstall.

The standalone Linux `uninstall.sh` only removes Wiz4rd Fr0g-owned installation artifacts from the current user profile and preserves personal settings. Linux release packages include `SHA256SUMS`; `install.sh` verifies the binary, icon and both lifecycle scripts before installation.

## Physical Windows package tests

The Windows VM harness executes the production flow:

install → install verification → production detection → production uninstall → post-uninstall verification.

Physical PASS is emitted only when all four verification points succeed. Tests must run on an isolated Windows executor and refuse to remove software that existed before the test.

The current execution host does not expose a Windows VM, Wine, Winget or a Windows CI executor. Therefore no physical package result is fabricated. The repository includes the GitHub Actions Windows test workflow and resumable local VM scripts for real execution.

## Root-cause classification

Physical failures are classified into explicit root-cause categories such as WrongPackageId, AmbiguousMatch, WrongRegistryMatch, WrongScope, UserScopeElevated, MachineScopeNotElevated, BrokenUninstallString, BrokenQuotedPath, MSIProductCode, MSIXStore, RebootRequired, RunningProcess, SystemComponent, UnsupportedAutomation, InstallerChanged, PackageUnavailable, DetectionFalsePositive and DetectionFalseNegative.

Known user-reported regressions for Firefox scope handling, Edge/WebView2 safety and quoted vendor uninstall commands are preserved as regression vectors. They are not marked physically verified until a real Windows VM result exists.

## Release gate

Run from a Git checkout with no uncommitted source changes:

`./scripts/release-gate.sh`

or on Windows:

`powershell -ExecutionPolicy Bypass -File .\scripts\release-gate.ps1`

Physical VM evidence is authenticated with HMAC-SHA256. Before executing physical tests and before running the release gate, set `WIZ4RDFR0G_EVIDENCE_HMAC_KEY` to the same secret value with at least 32 characters. The secret is not stored in the repository or release report. GitHub Actions expects it as the `WIZ4RDFR0G_EVIDENCE_HMAC_KEY` repository secret.

The pipeline first creates controlled release artifacts and `release/build_manifest.json`. The manifest binds the release to the application version, Git commit SHA, deterministic catalog fingerprint, evidence schema version and SHA256 of the exact Windows executable used for physical testing. It also records hashes for the Windows Setup, installer payload, Linux executable and Linux package. The Setup payload must match the Windows application artifact byte-for-byte.

Windows code signing is an explicit release state. Without signing configuration the manifest reports `unsigned`; a configured signing tool without verification reports `signed_unverified`; only a successful configured verification stage reports `signed_verified`. The build never claims a verified signature merely because a signing command was requested. Signing can be wired in with `WIZ4RDFR0G_SIGNTOOL`, JSON-array arguments in `WIZ4RDFR0G_SIGN_ARGS_JSON`, and optional verification arguments in `WIZ4RDFR0G_SIGN_VERIFY_ARGS_JSON`.

Every physical result records the build ID, Git commit, catalog fingerprint, exact tested executable SHA256, test-run ID, catalog-entry ID, machine ID and timestamps. Release-decision fields are signed before the result is written. Reboot-resume checkpoints are validated before they are trusted. Old-build, old-schema, wrong-catalog, wrong-artifact, unsigned, corrupt, duplicated and conflicting evidence is release-blocking rather than ignored.

The gate runs the controlled build, unit tests, static analysis, catalog audit, artifact-integrity validation, authenticated physical-evidence validation and coverage validation. Release is blocked if package resolution is unsafe, artifact identity does not match the manifest, evidence is invalid or stale, or any required catalog entry still lacks physical install/detect/uninstall/verify evidence.

The generated audit material includes `release/build_manifest.json`, `release/artifact_issues.json`, `release/evidence_issues.json`, `release/coverage.json`, `release/root_causes.json`, `release/regression_corpus.json` and `release/release_gate.json`.

## Publisher metadata

https://github.com/uhuabagoly
