//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	th32csSnapProcess              = 0x00000002
	processTerminate               = 0x0001
	processQueryLimitedInformation = 0x1000
	invalidHandleValue             = ^uintptr(0)
)

type processEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

type runningProcess struct {
	PID  uint32
	Path string
}

var (
	procCreateToolhelp32Snapshot   = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW            = kernel32.NewProc("Process32FirstW")
	procProcess32NextW             = kernel32.NewProc("Process32NextW")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procGetCurrentProcessId        = kernel32.NewProc("GetCurrentProcessId")
)

func processRootsFromState(idx int) ([]string, []string) {
	if idx < 0 || idx >= len(states) {
		return nil, nil
	}
	st := states[idx]
	var roots []string
	var exact []string
	if dir := cleanProcessPath(st.ActualInstallDir); safeProcessRoot(dir) {
		roots = appendUniquePath(roots, dir)
	}
	if icon := executablePathFromRegistryValue(st.InstalledDisplayIcon); icon != "" {
		icon = cleanProcessPath(icon)
		if safeProcessExecutable(icon) {
			exact = appendUniquePath(exact, icon)
			if dir := filepath.Dir(icon); safeProcessRoot(dir) {
				roots = appendUniquePath(roots, dir)
			}
		}
	}
	return roots, exact
}

func processRootsForMachineApp(app appDef) ([]string, []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	regs := registryPackagesByScope(scanRegistryPackages(), "machine")
	pkgs := listInstalledPackagesForScope(ctx, "machine")
	pkg, ok := resolveInstalledPackage(app, pkgs, regs)
	if !ok {
		return nil, nil
	}
	if pkg.RegistryKey == "" {
		if reg, found := resolveRegistryForInstalled(app, pkg, regs); found {
			pkg.InstallLocation = deriveRegistryInstallLocation(reg)
			pkg.DisplayIcon = reg.DisplayIcon
		}
	}
	var roots []string
	var exact []string
	if dir := cleanProcessPath(pkg.InstallLocation); safeProcessRoot(dir) {
		roots = appendUniquePath(roots, dir)
	}
	if icon := executablePathFromRegistryValue(pkg.DisplayIcon); icon != "" {
		icon = cleanProcessPath(icon)
		if safeProcessExecutable(icon) {
			exact = appendUniquePath(exact, icon)
			if dir := filepath.Dir(icon); safeProcessRoot(dir) {
				roots = appendUniquePath(roots, dir)
			}
		}
	}
	return roots, exact
}

func safeProcessRoot(path string) bool {
	path = cleanProcessPath(path)
	if path == "" {
		return false
	}
	lower := strings.ToLower(path)
	windows := cleanProcessPath(os.Getenv("WINDIR"))
	if windows != "" && (lower == strings.ToLower(windows) || strings.HasPrefix(lower, strings.ToLower(windows)+`\`)) {
		return false
	}
	if lower == `c:\` || lower == `c:\program files` || lower == `c:\program files (x86)` || lower == `c:\users` {
		return false
	}
	return true
}

func safeProcessExecutable(path string) bool {
	path = cleanProcessPath(path)
	return path != "" && strings.EqualFold(filepath.Ext(path), ".exe") && safeProcessRoot(filepath.Dir(path))
}

func cleanProcessPath(path string) string {
	path = strings.Trim(strings.TrimSpace(path), `"`)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func appendUniquePath(paths []string, path string) []string {
	for _, v := range paths {
		if strings.EqualFold(v, path) {
			return paths
		}
	}
	return append(paths, path)
}

func enumerateMatchingProcesses(roots, exact []string) []runningProcess {
	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snapshot == 0 || snapshot == invalidHandleValue {
		return nil
	}
	defer procCloseHandle.Call(snapshot)
	current, _, _ := procGetCurrentProcessId.Call()
	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	r, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	var out []runningProcess
	for r != 0 {
		if entry.ProcessID != 0 && uintptr(entry.ProcessID) != current {
			if path := processImagePath(entry.ProcessID); path != "" && pathMatchesKnownApp(path, roots, exact) {
				out = append(out, runningProcess{PID: entry.ProcessID, Path: path})
			}
		}
		entry.Size = uint32(unsafe.Sizeof(processEntry32{}))
		r, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
	return out
}

func processImagePath(pid uint32) string {
	h, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 || size == 0 {
		return ""
	}
	return filepath.Clean(syscall.UTF16ToString(buf[:size]))
}

func pathMatchesKnownApp(path string, roots, exact []string) bool {
	path = cleanProcessPath(path)
	for _, v := range exact {
		if strings.EqualFold(path, v) {
			return true
		}
	}
	lower := strings.ToLower(path)
	for _, root := range roots {
		r := strings.ToLower(cleanProcessPath(root))
		if r != "" && strings.HasPrefix(lower, r+`\`) {
			return true
		}
	}
	return false
}

func terminateKnownProcesses(roots, exact []string) (int, bool, error) {
	procs := enumerateMatchingProcesses(roots, exact)
	closed := 0
	needElevation := false
	for _, p := range procs {
		h, _, _ := procOpenProcess.Call(processTerminate|processQueryLimitedInformation, 0, uintptr(p.PID))
		if h == 0 {
			needElevation = true
			continue
		}
		r, _, callErr := procTerminateProcess.Call(h, 0)
		procCloseHandle.Call(h)
		if r == 0 {
			needElevation = true
			if callErr != nil && callErr != syscall.Errno(0) {
				continue
			}
			continue
		}
		closed++
	}
	if len(procs) > 0 && closed == 0 && needElevation {
		return 0, true, fmt.Errorf("a program folyamatai csak emelt jogosultsággal zárhatók be")
	}
	return closed, needElevation, nil
}

func runCloseRunningWorker(idx int) int {
	if idx < 0 || idx >= len(catalog) {
		return 2
	}
	roots, exact := processRootsForMachineApp(catalog[idx])
	if len(roots) == 0 && len(exact) == 0 {
		workerLog("WARN", catalog[idx].Name+": nem azonosítható biztonságosan bezárható programfolyamat.")
		return 3
	}
	closed, _, err := terminateKnownProcesses(roots, exact)
	if err != nil {
		workerLog("WARN", catalog[idx].Name+": folyamatbezárás sikertelen: "+err.Error())
		return 4
	}
	workerLog("INFO", fmt.Sprintf("%s: %d programfolyamat bezárva.", catalog[idx].Name, closed))
	return 0
}
