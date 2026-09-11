package main

import "strings"

const (
	uninstallCodeOK             = 0
	uninstallCodeFailed         = 1
	uninstallCodeNeedElevation  = 10
	uninstallCodeWrongElevation = 11
	uninstallCodeRunningProcess = 12
	uninstallCodeRebootRequired = 13
	uninstallCodeUnsupported    = 20
)

func uninstallExitSuccess(code int) bool {
	return code == 0 || code == 1641 || code == 3010
}

func uninstallNeedsElevation(out string) bool {
	v := strings.ToLower(out)
	needles := []string{"requires administrator", "administrator privileges are required", "access is denied", "access denied", "elevation required", "requires elevation", "0x800702e4"}
	for _, n := range needles {
		if strings.Contains(v, n) {
			return true
		}
	}
	return false
}

func uninstallWrongElevation(out string) bool {
	v := strings.ToLower(out)
	return strings.Contains(v, "cannot be uninstalled when running with administrator privileges") || strings.Contains(v, "installed for user scope cannot be uninstalled")
}

func uninstallRunningProcess(out string) bool {
	v := strings.ToLower(out)
	needles := []string{
		"close the application",
		"close the app",
		"application is running",
		"app is running",
		"program is running",
		"currently running",
		"is in use",
		"files are in use",
		"another instance is running",
		"please exit",
		"please close",
		"cannot continue while",
	}
	for _, n := range needles {
		if strings.Contains(v, n) {
			return true
		}
	}
	return false
}

type uninstallAttempt struct {
	Success        bool
	ExitCode       int
	Output         string
	Err            error
	RebootRequired bool
}

func makeUninstallAttempt(code int, out string, err error) uninstallAttempt {
	return uninstallAttempt{
		Success:        uninstallExitSuccess(code),
		ExitCode:       code,
		Output:         out,
		Err:            err,
		RebootRequired: code == 1641 || code == 3010,
	}
}

func classifyUninstallAttempt(attempt uninstallAttempt) int {
	if attempt.Success {
		if attempt.RebootRequired {
			return uninstallCodeRebootRequired
		}
		return uninstallCodeOK
	}
	if uninstallWrongElevation(attempt.Output) {
		return uninstallCodeWrongElevation
	}
	if uninstallNeedsElevation(attempt.Output) {
		return uninstallCodeNeedElevation
	}
	if uninstallRunningProcess(attempt.Output) {
		return uninstallCodeRunningProcess
	}
	return uninstallCodeFailed
}
