//go:build windows

package main

import "testing"

func TestDesktopIdentityDoesNotSelectSameNamedShellExtension(t *testing.T) {
	app := appDef{Name: "TortoiseSVN"}
	shell := installedPackage{Name: "TortoiseSVN", ID: `MSIX\3A48D7FC-AEE2-4CBC-91D1-0007951B8000`}
	if _, state := resolveInstalledPackageDetailed(app, []installedPackage{shell}, nil); state != installedResolveNone {
		t.Fatal("shell extension selected as desktop package")
	}
	main := installedPackage{Name: "TortoiseSVN 1.14.9.29743 (64 bit)", ID: "TortoiseSVN.TortoiseSVN"}
	got, state := resolveInstalledPackageDetailed(app, []installedPackage{shell, main}, nil)
	if state != installedResolveFound || got.ID != main.ID {
		t.Fatal("exact desktop identity not preferred")
	}
}
