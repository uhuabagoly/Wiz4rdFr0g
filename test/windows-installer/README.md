# Windows self-install lifecycle test

Run on an isolated Windows x64 test machine after producing `dist/Wiz4rd_Fr0g_Setup.exe`:

`powershell -ExecutionPolicy Bypass -File .\test\windows-installer\self-lifecycle.ps1`

The harness performs clean install, registry/shortcut validation, in-place upgrade with settings preservation, same-name foreign-process safety, quiet self-uninstall with disappearance verification, a second idempotent uninstall and partial-install cleanup.

The test is destructive only to the Wiz4rd Fr0g installation under Program Files, its standard uninstall registry key and its Start menu shortcut. Personal `settings.json` is expected to survive removal.

Cross-user UAC credential behavior still requires a manually controlled standard-user + alternate-admin Windows VM because hosted automation cannot faithfully emulate that interactive security boundary.
