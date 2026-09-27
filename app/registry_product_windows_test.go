//go:build windows

package main

import "testing"

func TestCorrelateWrapperRequiresSameMSIProductAndLocation(t *testing.T) {
	a := registryPackage{DisplayName: "Proton Mail Bridge", DisplayVersion: "3.27.0", Scope: "machine", InstallLocation: `C:\Program Files\Proton AG\Proton Mail Bridge`, WindowsInstaller: 1, RegistryKey: `HKLM\{ECCD475D-04B5-490D-B5C1-BE60AE77183C}`, UninstallString: `MsiExec.exe /I{ECCD475D-04B5-490D-B5C1-BE60AE77183C}`}
	b := a
	b.WindowsInstaller = 0
	b.RegistryKey = `HKLM\Proton Mail Bridge 3.27.0`
	b.UninstallString = `C:\ProgramData\Caphyon\Bridge-Installer.exe /i {ECCD475D-04B5-490D-B5C1-BE60AE77183C} AI_UNINSTALLER_CTP=1`
	if !sameRegisteredProduct(a, b) {
		t.Fatal("correlated product rejected")
	}
	b.InstallLocation = `C:\Other`
	if sameRegisteredProduct(a, b) {
		t.Fatal("different installation merged")
	}
	b.InstallLocation = a.InstallLocation
	b.UninstallString = `C:\Other\uninstall.exe`
	if sameRegisteredProduct(a, b) {
		t.Fatal("name-only match merged")
	}
}
