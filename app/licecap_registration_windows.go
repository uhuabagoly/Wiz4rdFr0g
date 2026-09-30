//go:build windows

package main

import (
	"context"
	"debug/pe"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LICEcap's publisher installer has no ARP entry: it stores the install root
// in HKLM\Software\LICEcap's default value and writes Uninstall.exe there.
// Adapt that real vendor registration; do not manufacture an ARP entry.
// https://github.com/justinfrankel/licecap/blob/main/licecap/installer.nsi
func scanLICEcapVendorRegistration() (registryPackage, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	const key = `HKEY_LOCAL_MACHINE\Software\LICEcap`
	code, out, err := runDirectProcess(ctx, "reg.exe", []string{"query", key, "/ve", "/reg:32"})
	if err != nil || code != 0 {
		return registryPackage{}, false
	}
	m := regexp.MustCompile(`(?m)^\s+.+?\s+REG_SZ\s+(.+?)\s*$`).FindStringSubmatch(out)
	if len(m) != 2 {
		return registryPackage{}, false
	}
	root := filepath.Clean(strings.TrimSpace(m[1]))
	if !licecapInstallRoot(root, os.Getenv("ProgramFiles(x86)")) {
		return registryPackage{}, false
	}
	bin, uninstaller := filepath.Join(root, "LICEcap.exe"), filepath.Join(root, "Uninstall.exe")
	file, err := pe.Open(bin)
	if err != nil {
		return registryPackage{}, false
	}
	file.Close()
	if !nsisUninstallerFile(uninstaller) {
		return registryPackage{}, false
	}
	return registryPackage{DisplayName: "LICEcap", InstallLocation: root, DisplayIcon: bin,
		UninstallString: `"` + uninstaller + `"`, RegistryKey: key, RegistryView: "/reg:32", Scope: "machine"}, true
}

func licecapInstallRoot(root, programFiles string) bool {
	return filepath.IsAbs(programFiles) && filepath.IsAbs(root) && strings.EqualFold(filepath.Clean(root), filepath.Join(programFiles, "LICEcap"))
}
