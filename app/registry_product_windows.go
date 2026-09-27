//go:build windows

package main

import "strings"

func registryForTruncatedPackage(pkg installedPackage, registrations []registryPackage) (registryPackage, registryMatchState) {
	name := strings.TrimSpace(pkg.Name)
	if !strings.HasSuffix(name, "…") || pkg.Version == "" || pkg.Scope == "" {
		return registryPackage{}, registryMatchNone
	}
	prefix := strings.TrimSuffix(name, "…")
	if len([]rune(prefix)) < 24 {
		return registryPackage{}, registryMatchNone
	}
	var found registryPackage
	count := 0
	for _, reg := range registrations {
		if reg.DisplayVersion == pkg.Version && reg.Scope == pkg.Scope && strings.HasPrefix(strings.ToLower(reg.DisplayName), strings.ToLower(prefix)) {
			found = reg
			count++
		}
	}
	if count == 0 {
		return registryPackage{}, registryMatchNone
	}
	if count != 1 {
		return registryPackage{}, registryMatchAmbiguous
	}
	return found, registryMatchFound
}

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
