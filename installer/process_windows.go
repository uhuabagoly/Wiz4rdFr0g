//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"wiz4rdfr0g.local/fullcatalog/internal/selfinstall"
)

const (
	th32csSnapProcess               = 0x00000002
	processTerminate                = 0x0001
	processQueryLimitedInfo         = 0x1000
	synchronize                     = 0x00100000
	waitTimeoutMilliseconds         = 5000
	invalidHandleValue      uintptr = ^uintptr(0)
)

var (
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = kernel32.NewProc("Process32FirstW")
	procProcess32NextW           = kernel32.NewProc("Process32NextW")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageW   = kernel32.NewProc("QueryFullProcessImageNameW")
	procTerminateProcess         = kernel32.NewProc("TerminateProcess")
)

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

// stopExactExecutable only terminates processes whose kernel-reported executable
// path exactly matches target. A same-named executable from another directory is
// intentionally ignored.
func stopExactExecutable(target string) error {
	snapshot, _, err := procCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snapshot == 0 || snapshot == invalidHandleValue {
		return fmt.Errorf("process snapshot failed: %v", err)
	}
	defer procCloseHandle.Call(snapshot)

	var entry processEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	r, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if r == 0 {
		return nil
	}
	selfPID := uint32(os.Getpid())
	for {
		if entry.ProcessID != 0 && entry.ProcessID != selfPID {
			path, handle, queryErr := executablePathForPID(entry.ProcessID, processTerminate|processQueryLimitedInfo|synchronize)
			if queryErr == nil && selfinstall.SameWindowsPath(path, target) {
				if rr, _, termErr := procTerminateProcess.Call(handle, 0); rr == 0 {
					procCloseHandle.Call(handle)
					return fmt.Errorf("failed to stop installed process %d (%s): %v", entry.ProcessID, path, termErr)
				}
				procWaitForSingleObject.Call(handle, waitTimeoutMilliseconds)
			}
			if handle != 0 {
				procCloseHandle.Call(handle)
			}
		}
		r, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if r == 0 {
			break
		}
	}
	return nil
}

func executablePathForPID(pid uint32, access uint32) (string, uintptr, error) {
	h, _, openErr := procOpenProcess.Call(uintptr(access), 0, uintptr(pid))
	if h == 0 {
		return "", 0, openErr
	}
	buf := make([]uint16, 32768)
	sz := uint32(len(buf))
	r, _, queryErr := procQueryFullProcessImageW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&sz)))
	if r == 0 {
		procCloseHandle.Call(h)
		return "", 0, queryErr
	}
	return syscall.UTF16ToString(buf[:sz]), h, nil
}
