//go:build linux

package main

import (
	"encoding/json"
	"testing"
)

func TestManagedPythonSelectionRejectsPrereleaseAndForeignSource(t *testing.T) {
	var rows []managedPython
	if err := json.Unmarshal([]byte(`[
	{"key":"preview","implementation":"cpython","version":"3.15.0rc1","version_parts":{"major":3,"minor":15,"patch":0},"url":"https://github.com/astral-sh/python-build-standalone/releases/download/preview"},
	{"key":"stable","implementation":"cpython","version":"3.14.1","version_parts":{"major":3,"minor":14,"patch":1},"url":"https://github.com/astral-sh/python-build-standalone/releases/download/stable"},
	{"key":"foreign","implementation":"cpython","version":"3.16.1","version_parts":{"major":3,"minor":16,"patch":1},"url":"https://example.invalid/python"}]`), &rows); err != nil {
		t.Fatal(err)
	}
	selected, ok := newestManagedPython(rows)
	if !ok || selected.Key != "stable" {
		t.Fatalf("wrong managed Python release: %+v", selected)
	}
}

func TestManagedPythonAstralMirror(t *testing.T) {
	const asset = "20260901/cpython-3.14.7%2B20260901-x86_64-unknown-linux-gnu-install_only_stripped.tar.gz"
	github := "https://github.com/astral-sh/python-build-standalone/releases/download/" + asset
	mirror := "https://releases.astral.sh/github/python-build-standalone/releases/download/" + asset
	if managedPythonSource(mirror) == "" || managedPythonSource(mirror) != managedPythonSource(github) {
		t.Fatal("official mirror must identify the same release artifact")
	}
	for _, address := range []string{"https://releases.astral.sh.evil.invalid/github/python-build-standalone/releases/download/" + asset, "https://example.invalid/" + asset} {
		if managedPythonSource(address) != "" {
			t.Fatal("foreign source accepted")
		}
	}
	var row managedPython
	row.Key, row.Version, row.Implementation, row.Variant, row.URL = "cpython-3.14.7-linux-x86_64-gnu", "3.14.7", "cpython", "default", mirror
	row.VersionParts.Major, row.VersionParts.Minor, row.VersionParts.Patch = 3, 14, 7
	if selected, ok := newestManagedPython([]managedPython{row}); !ok || selected.Key != row.Key {
		t.Fatal("current uv mirror release was not selected")
	}
}
