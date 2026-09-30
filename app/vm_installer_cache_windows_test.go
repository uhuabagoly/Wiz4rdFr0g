//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBitviseCacheRequiresExactVerifiedInstallerBytes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ProgramFiles(x86)", root)
	install := filepath.Join(root, "Bitvise SSH Client")
	cache := filepath.Join(install, "Updates", "BvSshClient-966.exe")
	if err := os.MkdirAll(filepath.Dir(cache), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, []byte("verified installer fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	n, digest, err := vmFileDigest(cache)
	if err != nil {
		t.Fatal(err)
	}
	r := registryPackage{DisplayName: "Bitvise SSH Client 9.66 (remove only)", DisplayVersion: "9.66", Scope: "machine", RegistryView: "/reg:32", RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\BvSshClient`, UninstallString: `"` + filepath.Join(install, "uninst.exe") + `" BvSshClient`}
	makeProof := func() vmFilesystemProof {
		return vmFilesystemProof{Registration: r, BinaryPaths: []string{cache, filepath.Join(install, "BvSsh.exe")}, BinariesPresent: true}
	}
	download := vmDownloadProof{Valid: true, Bytes: n, SHA256: digest, ExpectedSHA256: digest}
	p := makeProof()
	bad := download
	bad.ExpectedSHA256 = "different"
	vmSeparateBitviseInstallerCache(&p, bad)
	if len(p.BinaryPaths) != 2 || len(p.InstallerCaches) != 0 {
		t.Fatal("unverified download was accepted")
	}
	vmSeparateBitviseInstallerCache(&p, download)
	if len(p.BinaryPaths) != 1 || p.BinaryPaths[0] != filepath.Join(install, "BvSsh.exe") || len(p.InstallerCaches) != 1 {
		t.Fatalf("application runtime or cache proof lost: %+v", p)
	}
	if err := os.WriteFile(cache, []byte("changed installer fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := vmVerifyFilesystemRemoved(context.Background(), &p); err == nil {
		t.Fatal("changed retained cache passed removal verification")
	}
	p = makeProof()
	vmSeparateBitviseInstallerCache(&p, download)
	if len(p.BinaryPaths) != 2 || len(p.InstallerCaches) != 0 {
		t.Fatal("different bytes excluded from executable proof")
	}
}
