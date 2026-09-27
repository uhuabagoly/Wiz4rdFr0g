//go:build windows

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestChromiumSilentPreservesProfileAndExactTarget(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	root := filepath.Join(local, "Chromium", "Application")
	r := registryPackage{DisplayName: "Chromium", DisplayVersion: "153.0.8010.53", Scope: "user", InstallLocation: root, RegistryKey: `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Chromium`, UninstallString: `"` + filepath.Join(root, "153.0.8010.53", "Installer", "setup.exe") + `" --uninstall`}
	args, ok := observedVendorSilentArgs(appDef{Name: "Chromium"}, r)
	if !ok || !reflect.DeepEqual(args, []string{"--uninstall", "--force-uninstall"}) {
		t.Fatal("silent mode must not request profile deletion", args)
	}
	r.UninstallString += " --delete-profile"
	if _, ok := observedVendorSilentArgs(appDef{Name: "Chromium"}, r); ok {
		t.Fatal("unexpected command accepted")
	}
}

func TestObservedVendorSilentRejectsUnboundExecutable(t *testing.T) {
	root := t.TempDir()
	r := registryPackage{DisplayName: "GPT4All", DisplayVersion: "3.10.0", Scope: "user", InstallLocation: root, UninstallString: `"` + filepath.Join(root, "maintenancetool.exe") + `" --start-uninstaller`}
	if _, ok := observedVendorSilentArgs(appDef{Name: "GPT4All"}, r); !ok {
		t.Fatal("known Qt registration rejected")
	}
	r.InstallLocation = t.TempDir()
	if _, ok := observedVendorSilentArgs(appDef{Name: "GPT4All"}, r); ok {
		t.Fatal("different executable root accepted")
	}
	r = registryPackage{DisplayName: "SoapUI 5.10.0", DisplayVersion: "5.10.0", Scope: "machine", InstallLocation: root, RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\5517-2803-0637-4585`, UninstallString: `"` + filepath.Join(root, "uninstall.exe") + `"`}
	if _, ok := observedVendorSilentArgs(appDef{Name: "SoapUI"}, r); ok {
		t.Fatal("missing install4j evidence accepted")
	}
	if err := os.Mkdir(filepath.Join(root, ".install4j"), 0700); err != nil {
		t.Fatal(err)
	}
	if args, ok := observedVendorSilentArgs(appDef{Name: "SoapUI"}, r); !ok || len(args) != 1 || args[0] != "-q" {
		t.Fatal("known install4j registration rejected")
	}
	r.DisplayVersion = "6.0"
	if _, ok := observedVendorSilentArgs(appDef{Name: "SoapUI"}, r); ok {
		t.Fatal("unobserved version accepted")
	}
}
