//go:build windows

package main

import "testing"

func TestMultipassRetentionBindsExactMSI(t *testing.T) {
	const guid = "{A7AD2F65-C450-4440-9AD9-59591C1AA15E}"
	r := registryPackage{DisplayName: "Multipass", DisplayVersion: "1.16.4", Scope: "machine", WindowsInstaller: 1, RegistryView: "/reg:64", RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + guid, UninstallString: "MsiExec.exe /X" + guid}
	if !multipassPreserveDataRegistration(r) {
		t.Fatal("observed MSI rejected")
	}
	for _, change := range []func(*registryPackage){
		func(p *registryPackage) { p.DisplayVersion = "other" },
		func(p *registryPackage) { p.Scope = "user" },
		func(p *registryPackage) { p.RegistryKey += "other" },
		func(p *registryPackage) { p.WindowsInstaller = 0 },
	} {
		bad := r
		change(&bad)
		if multipassPreserveDataRegistration(bad) {
			t.Fatal("unbound MSI accepted")
		}
	}
}
