//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
)

// Code::Blocks 25.03 registers in HKCU even when its payload and NSIS
// uninstaller live in Program Files. A reduced-token uninstall removes the
// registration but leaves every executable. Elevate this exact vendor command
// through the existing worker flow while preserving its actual HKCU identity.
func codeBlocksMachinePayload(app appDef, pkg installedPackage, regs []registryPackage) (registryPackage, bool) {
	if app.Name != "Code::Blocks" || pkg.ID != "CodeBlocks.CodeBlocks" || pkg.Scope != "user" {
		return registryPackage{}, false
	}
	reg, ok := resolveRegistryForInstalled(app, pkg, regs)
	if !ok || reg.DisplayName != "CodeBlocks" || reg.DisplayVersion != "25.03" || reg.Scope != "user" || reg.RegistryView != "/reg:64" || !strings.EqualFold(reg.RegistryKey, `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\CodeBlocks`) {
		return registryPackage{}, false
	}
	root := os.Getenv("ProgramFiles")
	if !filepath.IsAbs(root) {
		return registryPackage{}, false
	}
	exe := filepath.Join(root, "CodeBlocks", "uninstall.exe")
	if !strings.EqualFold(strings.Trim(reg.UninstallString, `"`), exe) || !nsisUninstallerFile(exe) {
		return registryPackage{}, false
	}
	return reg, true
}
