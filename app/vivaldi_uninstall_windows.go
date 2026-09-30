//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
)

// https://help.vivaldi.com/desktop/install-update/how-to-uninstall-vivaldi/
// Bind the observed user installation before adding the publisher's silent
// switches. Keep the registered install directory and never delete the profile.
func vivaldiSilentArgs(app appDef, reg registryPackage) ([]string, bool) {
	if app.Name != "Vivaldi" || reg.DisplayName != "Vivaldi" || reg.DisplayVersion != "8.2.4133.80" || reg.Scope != "user" || reg.RegistryView != "/reg:64" || reg.WindowsInstaller != 0 || reg.QuietUninstallString != "" || !strings.EqualFold(reg.RegistryKey, `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Vivaldi`) {
		return nil, false
	}
	local := os.Getenv("LOCALAPPDATA")
	base := filepath.Join(local, "Vivaldi")
	root := filepath.Join(base, "Application")
	exe, args, err := splitRegisteredCommandRaw(reg.UninstallString)
	if err != nil || !filepath.IsAbs(local) || !strings.EqualFold(filepath.Clean(reg.InstallLocation), root) || !strings.EqualFold(exe, filepath.Join(root, reg.DisplayVersion, "Installer", "setup.exe")) || len(args) != 4 || args[0] != "--uninstall" || !strings.EqualFold(args[1], "--vivaldi-install-dir="+base) || args[2] != "--vivaldi-enable-crashlog-uploading=0" || args[3] != "--verbose-logging" {
		return nil, false
	}
	return append(args, "--vivaldi", "--force-uninstall"), true
}
