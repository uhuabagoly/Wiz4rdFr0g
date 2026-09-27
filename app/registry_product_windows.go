//go:build windows

package main

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Burn's explicit dependency edge binds its bundle to the MSI product; names
// alone do not. This key shape was observed in Tailscale and Moonlight evidence.
func burnDependencyKey(a, b registryPackage) string {
	if !strings.EqualFold(a.DisplayName, b.DisplayName) || a.DisplayVersion == "" || a.DisplayVersion != b.DisplayVersion || a.Scope != b.Scope {
		return ""
	}
	msi, bundle := a, b
	if a.WindowsInstaller == 0 && b.WindowsInstaller != 0 {
		msi, bundle = b, a
	}
	if msi.WindowsInstaller == 0 || bundle.WindowsInstaller != 0 || !strings.Contains(strings.ToLower(bundle.UninstallString), "/uninstall") {
		return ""
	}
	guid := regexp.MustCompile(`^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$`)
	product := registeredMSIProductCode(msi.UninstallString, msi.QuietUninstallString, msi.RegistryKey, true)
	bundleID := filepath.Base(bundle.RegistryKey)
	if !strings.EqualFold(product, filepath.Base(msi.RegistryKey)) {
		return ""
	}
	if !guid.MatchString(product) || !guid.MatchString(bundleID) || !regexp.MustCompile(`^[0-9]+(?:\.[0-9]+){1,3}$`).MatchString(msi.DisplayVersion) {
		return ""
	}
	hive := "HKEY_LOCAL_MACHINE"
	if msi.Scope == "user" {
		hive = "HKEY_CURRENT_USER"
	} else if msi.Scope != "machine" {
		return ""
	}
	return hive + `\SOFTWARE\Classes\Installer\Dependencies\` + product + "_v" + msi.DisplayVersion + `\Dependents\` + bundleID
}

func sameBurnProduct(a, b registryPackage) bool {
	key := burnDependencyKey(a, b)
	if key == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, view := range []string{"/reg:64", "/reg:32"} {
		code, _, err := runDirectProcess(ctx, "reg.exe", []string{"query", key, view})
		if err == nil && code == 0 {
			return true
		}
	}
	return false
}

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
