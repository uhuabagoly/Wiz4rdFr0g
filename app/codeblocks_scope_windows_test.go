//go:build windows

package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestCodeBlocksElevationNeedsExactProtectedVendor(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ProgramFiles", root)
	dir := filepath.Join(root, "CodeBlocks")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "uninstall.exe")
	data := make([]byte, 128)
	copy(data, "MZ")
	copy(data[68:], []byte{0xef, 0xbe, 0xad, 0xde, 'N', 'u', 'l', 'l', 's', 'o', 'f', 't', 'I', 'n', 's', 't'})
	binary.LittleEndian.PutUint32(data[64:], 1)
	binary.LittleEndian.PutUint32(data[84:], 128)
	if err := os.WriteFile(exe, data, 0600); err != nil {
		t.Fatal(err)
	}
	app := appDef{Name: "Code::Blocks"}
	pkg := installedPackage{ID: "CodeBlocks.CodeBlocks", Name: "CodeBlocks", Scope: "user"}
	reg := registryPackage{DisplayName: "CodeBlocks", DisplayVersion: "25.03", Scope: "user", RegistryView: "/reg:64", RegistryKey: `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\CodeBlocks`, UninstallString: exe}
	if _, ok := codeBlocksMachinePayload(app, pkg, []registryPackage{reg}); !ok {
		t.Fatal("observed vendor layout rejected")
	}
	other := reg
	other.UninstallString = filepath.Join(t.TempDir(), "uninstall.exe")
	if _, ok := codeBlocksMachinePayload(app, pkg, []registryPackage{other}); ok {
		t.Fatal("user-writable command accepted")
	}
	other = reg
	other.DisplayVersion = "26.01"
	if _, ok := codeBlocksMachinePayload(app, pkg, []registryPackage{other}); ok {
		t.Fatal("unobserved version accepted")
	}
	if err := os.WriteFile(exe, []byte("not an NSIS uninstaller"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := codeBlocksMachinePayload(app, pkg, []registryPackage{reg}); ok {
		t.Fatal("unverified executable accepted")
	}
}
