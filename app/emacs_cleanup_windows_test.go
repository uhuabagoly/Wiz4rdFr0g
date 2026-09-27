//go:build windows

package main

import "testing"

func TestEmacsOrphanRequiresRemovedVersionAndUninstaller(t *testing.T) {
	r := registryPackage{DisplayName: "GNU Emacs 31.1", DisplayVersion: "31.1", Scope: "machine", RegistryView: "/reg:64", RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\emacs-31.1`, UninstallString: `"C:\Program Files\Emacs\Uninstall-31.1.exe"`}
	missing := func(string) bool { return true }
	if !emacsOrphanEligible(r, `C:\Program Files`, missing) {
		t.Fatal("known orphan rejected")
	}
	for _, present := range []string{`C:\Program Files\Emacs\Uninstall-31.1.exe`, `C:\Program Files\Emacs\emacs-31.1`} {
		if emacsOrphanEligible(r, `C:\Program Files`, func(path string) bool { return path != present }) {
			t.Fatal("existing payload or uninstaller accepted")
		}
	}
	r.RegistryView = "/reg:32"
	if emacsOrphanEligible(r, `C:\Program Files`, missing) {
		t.Fatal("different registry view accepted")
	}
	r.RegistryView = "/reg:64"
	r.DisplayVersion = "32.1"
	if emacsOrphanEligible(r, `C:\Program Files`, missing) {
		t.Fatal("different version accepted")
	}
}
