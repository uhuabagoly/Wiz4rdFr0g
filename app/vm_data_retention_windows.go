//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type vmPreservedDataProof struct {
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
	Preserved bool   `json:"preserved"`
}

func vmVerifyDataRetention(proof *vmFilesystemProof) error {
	for i := range proof.PreservedData {
		p := &proof.PreservedData[i]
		n, digest, err := vmFileDigest(p.Path)
		p.Preserved = err == nil && n == p.Bytes && digest == p.SHA256
		if !p.Preserved {
			return fmt.Errorf("vendor uninstall removed or altered the data-retention witness: %s", p.Path)
		}
	}
	return nil
}

// Only the physical VM harness creates this new owned witness, never a user
// configuration or VM disk. The vendor's data-purge action would delete it.
func vmCaptureDataRetention(name string, proof *vmFilesystemProof) error {
	if name != "Multipass" {
		return nil
	}
	if !multipassPreserveDataRegistration(proof.Registration) || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" {
		return fmt.Errorf("data retention probe requires the bound Multipass MSI on a disposable runner")
	}
	base := os.Getenv("ProgramData")
	if !filepath.IsAbs(base) {
		return fmt.Errorf("missing machine data directory")
	}
	// The installed vendor creates this directory; do not create it to mask a
	// missing installation or overwrite any existing user file.
	f, err := os.CreateTemp(filepath.Join(base, "Multipass"), "Wiz4rdFr0g-retention-*.txt")
	if err != nil {
		return err
	}
	path := f.Name()
	_, writeErr := f.WriteString("Wiz4rd Fr0g physical data-preservation witness\n")
	closeErr := f.Close()
	if writeErr != nil {
		os.Remove(path)
		return writeErr
	}
	if closeErr != nil {
		os.Remove(path)
		return closeErr
	}
	n, hash, err := vmFileDigest(path)
	if err != nil {
		return err
	}
	proof.PreservedData = append(proof.PreservedData, vmPreservedDataProof{Path: path, Bytes: n, SHA256: hash})
	return nil
}
