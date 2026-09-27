//go:build windows

package main

import "testing"

func TestSQLiteStudioUnattendedIdentity(t *testing.T) {
	app := appDef{Name: "SQLiteStudio"}
	reg := registryPackage{DisplayName: "SQLiteStudio", DisplayVersion: "3.4.21", Scope: "machine", InstallLocation: `C:\Program Files/SQLiteStudio`, RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\SQLiteStudio`, UninstallString: `"C:\Program Files\SQLiteStudio\uninstall.exe"`}
	if !sqliteStudioUnattended(app, reg) {
		t.Fatal("observed registration rejected")
	}
	for _, change := range []func(*registryPackage){
		func(r *registryPackage) { r.DisplayVersion = "3.5.0" },
		func(r *registryPackage) { r.UninstallString = `"C:\Other\uninstall.exe"` },
		func(r *registryPackage) { r.RegistryKey += "Other" },
		func(r *registryPackage) { r.Scope = "user" },
		func(r *registryPackage) { r.WindowsInstaller = 1 },
		func(r *registryPackage) { r.QuietUninstallString = "vendor-command" },
	} {
		other := reg
		change(&other)
		if sqliteStudioUnattended(app, other) {
			t.Fatal("unverified registration accepted")
		}
	}
}
