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
