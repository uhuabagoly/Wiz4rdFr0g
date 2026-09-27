//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// Python's per-user Burn registration can own an all-users interpreter.
// Require both the machine PEP 514 payload and a valid publisher signature
// before requesting elevation of this otherwise user-scoped target.
func protectedUserPayload(ctx context.Context, app appDef, pkg installedPackage, regs []registryPackage) (registryPackage, bool) {
	if reg, ok := codeBlocksMachinePayload(app, pkg, regs); ok {
		return reg, true
	}
	if (app.Name != "Python 3" && app.Name != "Python") || pkg.ID != "Python.Python.3.13" || pkg.Scope != "user" {
		return registryPackage{}, false
	}
	reg, ok := resolveRegistryForInstalled(app, pkg, regs)
	if !ok || reg.DisplayName != "Python 3.13.15 (64-bit)" || reg.DisplayVersion != "3.13.15150.0" || reg.Scope != "user" || reg.WindowsInstaller != 0 {
		return registryPackage{}, false
	}
	if !strings.EqualFold(reg.RegistryKey, `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{bdda61aa-57d0-45c8-ae92-fa67d560e32f}`) {
		return registryPackage{}, false
	}
	exe, args, err := splitRegisteredCommand(reg.QuietUninstallString)
	if err != nil || len(args) != 2 || args[0] != "/uninstall" || args[1] != "/quiet" || !strings.EqualFold(filepath.Base(exe), "python-3.13.15-amd64.exe") {
		return registryPackage{}, false
	}
	root := os.Getenv("ProgramFiles")
	if !filepath.IsAbs(root) {
		return registryPackage{}, false
	}
	path, err := registeredPython313Machine(ctx)
	if err != nil || !strings.EqualFold(path, filepath.Join(root, "Python313", "python.exe")) {
		return registryPackage{}, false
	}
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return registryPackage{}, false
	}
	quoted := "'" + strings.ReplaceAll(exe, "'", "''") + "'"
	script := "$s=Get-AuthenticodeSignature -LiteralPath " + quoted + "; if($s.Status -eq 'Valid' -and $s.SignerCertificate.Subject -match '(^|, )CN=Python Software Foundation(,|$)'){ 'VERIFIED_PSF' }else{ exit 1 }"
	code, out, err := runDirectProcess(ctx, "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script})
	if err != nil || code != 0 || strings.TrimSpace(out) != "VERIFIED_PSF" {
		return registryPackage{}, false
	}
	reg.QuietUninstallString += " InstallAllUsers=1"
	return reg, true
}
