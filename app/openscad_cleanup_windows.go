//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OpenSCAD 2021.01 physically removes its executables but can leave its
// 64-bit ARP key. Repair only this observed orphan, after a successful vendor
// uninstall. Never remove application files or accept an inaccessible file.
func openSCADOrphanEligible(reg registryPackage, programFiles string, absent func(string) bool) bool {
	const key = `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\OpenSCAD`
	if reg.DisplayName != "OpenSCAD (remove only)" || reg.DisplayVersion != "2021.01" || reg.RegistryView != "/reg:64" || reg.WindowsInstaller != 0 || !strings.EqualFold(reg.RegistryKey, key) || !filepath.IsAbs(programFiles) {
		return false
	}
	root := filepath.Join(programFiles, "OpenSCAD")
	exe := filepath.Join(root, "Uninstall.exe")
	if !strings.EqualFold(strings.Trim(reg.UninstallString, `"`), exe) {
		return false
	}
	for _, name := range []string{"Uninstall.exe", "openscad.exe", "openscad.com"} {
		if !absent(filepath.Join(root, name)) {
			return false
		}
	}
	return true
}

func repairOpenSCADOrphan(app appDef) bool {
	if app.Name != "OpenSCAD" {
		return false
	}
	absent := func(path string) bool { _, err := os.Stat(path); return os.IsNotExist(err) }
	var matches []registryPackage
	for _, reg := range scanRegistryPackages() {
		if openSCADOrphanEligible(reg, os.Getenv("ProgramFiles"), absent) {
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
		workerLog("WARN", "OpenSCAD orphan registration repair failed: "+compactLog(out))
		return false
	}
	workerLog("REPAIR", "OpenSCAD 2021.01 vendor uninstall removed both application entry points and its uninstaller; removed its exact orphaned 64-bit ARP key.")
	return true
}
