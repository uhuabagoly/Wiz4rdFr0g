//go:build windows

package main

import (
	"os"
	"path/filepath"
)

// Bitvise documents running a copy from a temporary directory so the original
// installer process and working directory do not obstruct removal or return
// before the child uninstaller finishes. Only our temporary copy is cleaned up.
// https://bitvise.com/ssh-server-guide-installing
func copyBitviseUninstaller(exe string) (string, func(), error) {
	data, err := os.ReadFile(exe)
	if err != nil {
		return "", nil, err
	}
	dir, err := os.MkdirTemp("", "Wiz4rdFr0g-Bitvise-uninstall-")
	if err != nil {
		return "", nil, err
	}
	copyPath := filepath.Join(dir, "uninst.exe")
	cleanup := func() {
		os.Remove(copyPath)
		os.Remove(dir)
	}
	if err := os.WriteFile(copyPath, data, 0700); err != nil {
		cleanup()
		return "", nil, err
	}
	return copyPath, cleanup, nil
}
