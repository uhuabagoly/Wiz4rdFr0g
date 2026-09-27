//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Emacs 31.1 sets registry view 64 during installation but omits it from
// the NSIS uninstall section. Repair only its proven empty registration.
// https://github.com/emacs-mirror/emacs/blob/emacs-31.1/admin/nt/dist-build/emacs.nsi
func emacsOrphanEligible(reg registryPackage, programFiles string, absent func(string) bool) bool {
	if reg.DisplayName != "GNU Emacs 31.1" || reg.DisplayVersion != "31.1" || reg.Scope != "machine" || reg.RegistryView != "/reg:64" || reg.WindowsInstaller != 0 || !filepath.IsAbs(programFiles) || !strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\emacs-31.1`) {
		return false
	}
	root := filepath.Join(programFiles, "Emacs")
	exe := filepath.Join(root, "Uninstall-31.1.exe")
	return strings.EqualFold(strings.Trim(reg.UninstallString, `"`), exe) && absent(exe) && absent(filepath.Join(root, "emacs-31.1"))
}

func repairEmacsOrphan(app appDef) bool {
	if app.Name != "Emacs" {
		return false
	}
	absent := func(path string) bool { _, err := os.Stat(path); return os.IsNotExist(err) }
	var matches []registryPackage
	for _, reg := range scanRegistryPackages() {
		if emacsOrphanEligible(reg, os.Getenv("ProgramFiles"), absent) {
			matches = append(matches, reg)
		}
	}
	if len(matches) != 1 {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reg := matches[0]
	code, out, err := runDirectProcess(ctx, "reg.exe", []string{"delete", reg.RegistryKey, "/f", reg.RegistryView})
	if err != nil || code != 0 {
		workerLog("WARN", "Emacs orphan registration repair failed: "+compactLog(out))
		return false
	}
	workerLog("REPAIR", "Emacs 31.1 vendor uninstall removed the entire version directory and uninstaller; removed its exact orphaned 64-bit ARP key.")
	return true
}
