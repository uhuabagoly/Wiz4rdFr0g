//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Chromium's documented vendor status differs from MSI/ordinary process codes.
// Preserve the original code and require the exact observed vendor identity.
// https://github.com/chromium/chromium/blob/main/chrome/installer/util/util_constants.h
func chromiumUninstallAttempt(app appDef, reg registryPackage, code int, out string, err error) uninstallAttempt {
	attempt := makeUninstallAttempt(code, out, err)
	if app.Name != "Brave" && app.Name != "Chromium" {
		return attempt
	}
	if _, ok := observedVendorSilentArgs(app, reg); !ok {
		return attempt
	}
	if code == 19 || code == 29 { // UNINSTALL_SUCCESSFUL / UNINSTALL_REQUIRES_REBOOT
		attempt.Success = true
		attempt.RebootRequired = code == 29
		attempt.Err = nil
		workerLog("INFO", fmt.Sprintf("%s: documented Chromium uninstall status=%d reboot=%v; removal verification still required", app.Name, code, attempt.RebootRequired))
	}
	return attempt
}

// Chromium moves the running setup.exe to TEMP before removing its directory.
// A cross-volume TEMP prevents that move. Keep both TEMP and the working
// directory outside Application, on the same volume, for this child only.
// https://github.com/chromium/chromium/blob/main/chrome/installer/setup/uninstall.cc
func runChromiumVendorUninstaller(ctx context.Context, exe string, args []string, root string) (int, string, error) {
	if !filepath.IsAbs(root) || !strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(exe)) {
		return -1, "", fmt.Errorf("browser uninstaller volume is not bound to installation")
	}
	dir, err := os.MkdirTemp(filepath.Dir(root), "Wiz4rdFr0g-uninstall-")
	if err != nil {
		return -1, "", err
	}
	// Remove only our empty directory. Vendor-owned files are left to the
	// vendor's own self-cleanup; never recursively delete application data.
	defer os.Remove(dir)
	env := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, "TEMP") && !strings.EqualFold(key, "TMP") {
			env = append(env, entry)
		}
	}
	env = append(env, "TEMP="+dir, "TMP="+dir)
	return runDirectProcessEnvironment(ctx, exe, args, "", dir, env)
}
