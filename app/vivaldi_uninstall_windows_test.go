//go:build windows

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestVivaldiSilentBindsObservedRegistration(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	base := filepath.Join(local, "Vivaldi")
	root := filepath.Join(base, "Application")
	r := registryPackage{DisplayName: "Vivaldi", DisplayVersion: "8.2.4133.80", Scope: "user", RegistryView: "/reg:64", RegistryKey: `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Vivaldi`, InstallLocation: root, UninstallString: `"` + filepath.Join(root, "8.2.4133.80", "Installer", "setup.exe") + `" --uninstall --vivaldi-install-dir="` + base + `" --vivaldi-enable-crashlog-uploading=0 --verbose-logging`}
	a := appDef{Name: "Vivaldi"}
	args, ok := observedVendorSilentArgs(a, r)
	if !ok || len(args) != 6 || args[4] != "--vivaldi" || args[5] != "--force-uninstall" || strings.Contains(strings.Join(args, " "), "delete-profile") {
		t.Fatal("publisher silent mode rejected", args)
	}
	for _, change := range []func(*registryPackage){func(r *registryPackage) { r.DisplayVersion = "8.3" }, func(r *registryPackage) { r.InstallLocation = filepath.Dir(root) }, func(r *registryPackage) { r.RegistryKey += "Other" }, func(r *registryPackage) { r.UninstallString += " --delete-profile" }, func(r *registryPackage) { r.Scope = "machine" }} {
		bad := r
		change(&bad)
		if _, ok := observedVendorSilentArgs(a, bad); ok {
			t.Fatal("unbound Vivaldi accepted")
		}
	}
}
