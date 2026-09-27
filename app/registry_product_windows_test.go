//go:build windows

package main

import "testing"

func TestTruncatedInventoryNeedsUniqueSameVersionAndScope(t *testing.T) {
	pkg := installedPackage{Name: "Eclipse Temurin JDK with Hotspot 8u504-…", Version: "8.0.504.1", Scope: "machine"}
	reg := registryPackage{DisplayName: "Eclipse Temurin JDK with Hotspot 8u504-b01 (x64)", DisplayVersion: "8.0.504.1", Scope: "machine"}
	if _, s := registryForTruncatedPackage(pkg, []registryPackage{reg}); s != registryMatchFound {
		t.Fatal("captured registry identity rejected")
	}
	if _, s := registryForTruncatedPackage(pkg, []registryPackage{reg, reg}); s != registryMatchAmbiguous {
		t.Fatal("ambiguous identity accepted")
	}
	reg.DisplayVersion = "8.0.503.1"
	if _, s := registryForTruncatedPackage(pkg, []registryPackage{reg}); s != registryMatchNone {
		t.Fatal("different version accepted")
	}
}

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
