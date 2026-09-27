//go:build windows

package main

import "strings"

// Advanced Installer may register both its wrapper and the underlying MSI.
// Merge identity only when both records explicitly refer to that same product,
// version, scope and installation directory; a matching name alone is unsafe.
func sameRegisteredProduct(a, b registryPackage) bool {
	if !strings.EqualFold(a.DisplayName, b.DisplayName) || a.DisplayVersion == "" || a.DisplayVersion != b.DisplayVersion || a.Scope != b.Scope || a.InstallLocation == "" || !strings.EqualFold(strings.TrimRight(a.InstallLocation, `\`), strings.TrimRight(b.InstallLocation, `\`)) {
		return false
	}
	var msi, wrapper registryPackage
	if a.WindowsInstaller != 0 && b.WindowsInstaller == 0 {
		msi, wrapper = a, b
	} else if b.WindowsInstaller != 0 && a.WindowsInstaller == 0 {
		msi, wrapper = b, a
	} else {
		return false
	}
	code := registeredMSIProductCode(msi.UninstallString, msi.QuietUninstallString, msi.RegistryKey, true)
	other := extractMSIProductCode(wrapper.UninstallString)
	return code != "" && strings.EqualFold(code, other)
}
