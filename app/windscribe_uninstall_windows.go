//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// The exact observed vendor's first phase returns a PID, not an exit status.
// Wait only on a held process handle whose executable matches the original
// uninstaller bytes. Never turn arbitrary nonzero exit codes into success.
// https://github.com/Windscribe/Desktop-App/blob/v2.24.13/src/installer/gui/windows/uninstaller/copy_and_run.cpp
func runWindscribeVendorUninstaller(ctx context.Context, exe string, args []string) (int, string, error) {
	original, err := os.ReadFile(exe)
	if err != nil {
		return -1, "", err
	}
	code, out, runErr := runDirectProcess(ctx, exe, args)
	if code <= 0 || uint64(code) >= 0xffffffff || ctx.Err() != nil {
		return code, out, runErr
	}
	h, _, openErr := procOpenProcess.Call(processQueryLimitedInformation|0x00100000, 0, uintptr(code)) // SYNCHRONIZE
	if h == 0 {
		return code, out, fmt.Errorf("vendor child PID %d could not be opened: %v", code, openErr)
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	ok, _, queryErr := procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ok == 0 || size == 0 {
		return code, out, fmt.Errorf("vendor child image could not be verified: %v", queryErr)
	}
	child := syscall.UTF16ToString(buf[:size])
	data, readErr := os.ReadFile(child)
	if readErr != nil || !windscribeChildMatches(original, child, data) {
		return code, out, fmt.Errorf("vendor child image differs from registered uninstaller: %v", readErr)
	}
	workerLog("INFO", fmt.Sprintf("Windscribe: first phase returned PID %d; verified identical vendor child, waiting for completion", code))
	for {
		if err := ctx.Err(); err != nil {
			return -1, out, err
		}
		wait, _, waitErr := procWaitForSingleObject.Call(h, 200)
		if wait == WAIT_TIMEOUT {
			continue
		}
		if wait != WAIT_OBJECT_0 {
			return -1, out, fmt.Errorf("vendor child wait failed: %v", waitErr)
		}
		var final uint32
		ok, _, exitErr := procGetExitCodeProcess.Call(h, uintptr(unsafe.Pointer(&final)))
		if ok == 0 {
			return -1, out, fmt.Errorf("vendor child exit status unavailable: %v", exitErr)
		}
		if final != 0 {
			return int(final), out, fmt.Errorf("vendor child exit status %d", final)
		}
		// Production detection and independent physical removal proof still run.
		return 0, out, nil
	}
}

func windscribeChildMatches(original []byte, child string, data []byte) bool {
	return len(original) > 0 && filepath.IsAbs(child) && strings.EqualFold(filepath.Base(child), "uninstall.exe") && sha256.Sum256(original) == sha256.Sum256(data)
}
