//go:build windows

package main

import (
	"path/filepath"
	"strings"
)

// The 3.4.21 publisher build uses InstallBuilder, whose uninstaller accepts
// --mode unattended. Limit this override to the observed vendor registration.
// https://github.com/pawelsalawa/sqlitestudio/blob/3.4.21/.github/workflows/win64_release.yml
// https://releases.installbuilder.com/installbuilder/docs/installbuilder-userguide.html
func sqliteStudioUnattended(app appDef, reg registryPackage) bool {
	if app.Name != "SQLiteStudio" || reg.DisplayName != "SQLiteStudio" || reg.DisplayVersion != "3.4.21" || reg.WindowsInstaller != 0 || reg.Scope != "machine" {
		return false
	}
	if !strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\SQLiteStudio`) {
		return false
	}
	exe, args, err := splitRegisteredCommandRaw(reg.UninstallString)
	root := filepath.Clean(reg.InstallLocation)
	return err == nil && len(args) == 0 && reg.QuietUninstallString == "" && filepath.IsAbs(root) && strings.EqualFold(filepath.Base(root), "SQLiteStudio") && strings.EqualFold(exe, filepath.Join(root, "uninstall.exe"))
}
