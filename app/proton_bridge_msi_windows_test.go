//go:build windows

package main

import (
	"path/filepath"
	"testing"
)

func TestProtonBridgeMSIRequiresObservedProductBinding(t *testing.T) {
	pf := t.TempDir()
	t.Setenv("ProgramFiles", pf)
	const guid = "{ECCD475D-04B5-490D-B5C1-BE60AE77183C}"
	r := registryPackage{DisplayName: "Proton Mail Bridge", DisplayVersion: "3.27.0", Scope: "machine", WindowsInstaller: 1, RegistryView: "/reg:64", RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + guid, UninstallString: "MsiExec.exe /I" + guid, InstallLocation: filepath.Join(pf, "Proton AG", "Proton Mail Bridge")}
	a := appDef{Name: "Proton Mail Bridge"}
	if !protonBridgeDirectMSI(a, r) {
		t.Fatal("observed MSI rejected")
	}
	for _, change := range []func(*registryPackage){
		func(p *registryPackage) { p.InstallLocation = pf },
		func(p *registryPackage) { p.DisplayVersion = "other" },
		func(p *registryPackage) { p.RegistryKey += "other" },
		func(p *registryPackage) { p.UninstallString = "MsiExec.exe /I{11111111-1111-1111-1111-111111111111}" },
	} {
		bad := r
		change(&bad)
		if protonBridgeDirectMSI(a, bad) {
			t.Fatal("unbound product accepted")
		}
	}
}
