//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
)

func protonBridgeDirectMSI(app appDef, reg registryPackage) bool {
	const guid = "{ECCD475D-04B5-490D-B5C1-BE60AE77183C}"
	pf := os.Getenv("ProgramFiles")
	return app.Name == "Proton Mail Bridge" && reg.DisplayName == "Proton Mail Bridge" && reg.DisplayVersion == "3.27.0" && reg.WindowsInstaller == 1 && reg.Scope == "machine" && reg.RegistryView == "/reg:64" && filepath.IsAbs(pf) &&
		strings.EqualFold(filepath.Clean(reg.InstallLocation), filepath.Join(pf, "Proton AG", "Proton Mail Bridge")) &&
		strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\`+guid) &&
		strings.EqualFold(registeredMSIProductCode(reg.UninstallString, reg.QuietUninstallString, reg.RegistryKey, true), guid)
}
