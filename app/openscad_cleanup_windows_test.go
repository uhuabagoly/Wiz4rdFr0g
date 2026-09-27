//go:build windows

package main

import "testing"

func TestOpenSCADOrphanEligibility(t *testing.T) {
	reg := registryPackage{DisplayName: "OpenSCAD (remove only)", DisplayVersion: "2021.01", RegistryView: "/reg:64", RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\OpenSCAD`, UninstallString: `C:\Program Files\OpenSCAD\Uninstall.exe`}
	absent := func(string) bool { return true }
	if !openSCADOrphanEligible(reg, `C:\Program Files`, absent) {
		t.Fatal("observed orphan should be eligible")
	}
	if openSCADOrphanEligible(reg, `C:\Program Files`, func(string) bool { return false }) {
		t.Fatal("present or inaccessible payload must block repair")
	}
	for _, mutate := range []func(*registryPackage){
		func(r *registryPackage) { r.DisplayVersion = "2026.01" },
		func(r *registryPackage) { r.RegistryView = "/reg:32" },
		func(r *registryPackage) { r.RegistryKey += "Other" },
		func(r *registryPackage) { r.UninstallString = `D:\Other\Uninstall.exe` },
		func(r *registryPackage) { r.WindowsInstaller = 1 },
	} {
		other := reg
		mutate(&other)
		if openSCADOrphanEligible(other, `C:\Program Files`, absent) {
			t.Fatal("different registration must not be repaired")
		}
	}
}
