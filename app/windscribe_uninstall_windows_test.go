//go:build windows

package main

import (
	"path/filepath"
	"testing"
)

func TestWindscribeChildIdentityRequiresExactBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "uninstall.exe")
	if !windscribeChildMatches([]byte("vendor"), path, []byte("vendor")) {
		t.Fatal("identical vendor child rejected")
	}
	for _, tc := range []struct{ original, path, data string }{
		{"vendor", path, "changed"}, {"vendor", "uninstall.exe", "vendor"},
		{"vendor", filepath.Join(t.TempDir(), "other.exe"), "vendor"}, {"", path, ""},
	} {
		if windscribeChildMatches([]byte(tc.original), tc.path, []byte(tc.data)) {
			t.Fatal("unbound child accepted")
		}
	}
}

func TestWindscribeSilentRequiresObservedRegistration(t *testing.T) {
	pf := t.TempDir()
	t.Setenv("ProgramFiles", pf)
	root := filepath.Join(pf, "Windscribe")
	r := registryPackage{DisplayName: "Windscribe", DisplayVersion: "2.24.13", Scope: "machine", RegistryView: "/reg:64", InstallLocation: root, UninstallString: `"` + filepath.Join(root, "uninstall.exe") + `"`, RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{fa690e90-ddb0-4f0c-b3f1-136c084e5fc7}_is1`}
	a, ok := observedVendorSilentArgs(appDef{Name: "Windscribe"}, r)
	if !ok || len(a) != 1 || a[0] != "/VERYSILENT" {
		t.Fatal("observed vendor silent mode missing")
	}
	r.DisplayVersion = "other"
	if _, ok := observedVendorSilentArgs(appDef{Name: "Windscribe"}, r); ok {
		t.Fatal("unobserved version accepted")
	}
}
