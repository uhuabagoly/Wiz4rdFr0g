//go:build windows

package main

import "testing"

func TestVMProcessRepairRootRequiresUnambiguousPayloadRegistration(t *testing.T) {
	app := appDef{Name: "Greenshot"}
	pkg := installedPackage{Name: "Greenshot 1.3.315", ID: "Greenshot.Greenshot", Scope: "user"}
	reg := registryPackage{DisplayName: "Greenshot 1.3.315", Scope: "user", RegistryKey: `HKCU\Greenshot`, UninstallString: `"C:\Users\runner\AppData\Local\Programs\Greenshot\unins000.exe"`}
	d := vmDetectedFrom(app, pkg, "user", []registryPackage{reg})
	if d.InstallLocation != `C:\Users\runner\AppData\Local\Programs\Greenshot` {
		t.Fatalf("missing repair root: %q", d.InstallLocation)
	}
	other := reg
	other.RegistryKey = `HKCU\Other`
	if d = vmDetectedFrom(app, pkg, "user", []registryPackage{reg, other}); d.InstallLocation != "" {
		t.Fatal("ambiguous registry supplied a process-termination root")
	}
	reg.UninstallString = `"C:\ProgramData\Package Cache\{ABC}\setup.exe" /uninstall`
	if d = vmDetectedFrom(app, pkg, "user", []registryPackage{reg}); d.InstallLocation != "" {
		t.Fatal("installer cache supplied a process-termination root")
	}
	if !uninstallRunningProcess("Uninstall has detected that Greenshot is currently running. Please close all instances of it now") {
		t.Fatal("vendor running-app reason unrecognized")
	}
}
