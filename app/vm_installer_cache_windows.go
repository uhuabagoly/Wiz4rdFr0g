//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type vmInstallerCacheProof struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
	Reason string `json:"reason"`
}

// The vendor leaves its downloaded installer under Updates. Do not mistake an
// exact copy of the independently verified HTTP installer for an application
// executable. No name-only or directory-wide exclusion is permitted.
func vmSeparateBitviseInstallerCache(proof *vmFilesystemProof, download vmDownloadProof) {
	if !download.Valid || download.Bytes <= 0 || len(download.SHA256) != 64 || download.SHA256 != download.ExpectedSHA256 || len(proof.BinaryPaths) < 2 {
		return
	}
	if _, ok := observedVendorSilentArgs(appDef{Name: "Bitvise SSH Client"}, proof.Registration); !ok {
		return
	}
	exe, _, err := splitRegisteredCommandRaw(proof.Registration.UninstallString)
	if err != nil {
		return
	}
	cache := filepath.Join(filepath.Dir(exe), "Updates", "BvSshClient-966.exe")
	for i, path := range proof.BinaryPaths {
		if !strings.EqualFold(path, cache) {
			continue
		}
		n, digest, err := vmFileDigest(path)
		if err != nil || n != download.Bytes || digest != download.SHA256 {
			return
		}
		proof.InstallerCaches = append(proof.InstallerCaches, vmInstallerCacheProof{Path: path, SHA256: digest, Bytes: n, Reason: "byte-identical copy of independently verified HTTP installer"})
		proof.BinaryPaths = append(proof.BinaryPaths[:i], proof.BinaryPaths[i+1:]...)
		return
	}
}

func vmFileDigest(path string) (int64, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	return n, hex.EncodeToString(h.Sum(nil)), err
}
