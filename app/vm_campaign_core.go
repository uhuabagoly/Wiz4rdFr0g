package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func isTransientPackageManagerFailure(text string) bool {
	v := strings.ToLower(strings.TrimSpace(text))
	needles := []string{
		"temporarily unavailable",
		"temporary failure",
		"source data is missing",
		"failed when opening source",
		"failed to update source",
		"source reset",
		"network connection",
		"network is unreachable",
		"name resolution",
		"dns",
		"timed out",
		"timeout",
		"0x8a15000f",
		"0x80072ee2",
		"0x80072efd",
		"0x801901f7",
		"http 429",
		"too many requests",
		"service unavailable",
		"gateway timeout",
	}
	for _, n := range needles {
		if strings.Contains(v, n) {
			return true
		}
	}
	return false
}

func campaignDurationFromEnv(name string, fallback, min, max time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	minutes, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	d := time.Duration(minutes) * time.Minute
	if d < min {
		return min
	}
	if d > max {
		return max
	}
	return d
}

func campaignRetryLimitFromEnv(name string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	if n < 0 {
		return 0
	}
	if n > 2 {
		return 2
	}
	return n
}

func shouldRetryUninstall(code int, installedAfter bool, attempt, maxAttempts int) bool {
	if attempt >= maxAttempts {
		return false
	}
	if code == uninstallCodeRebootRequired || code == uninstallCodeNeedElevation || code == uninstallCodeWrongElevation || code == uninstallCodeUnsupported {
		return false
	}
	return code == uninstallCodeFailed || code == uninstallCodeRunningProcess || (code == uninstallCodeOK && installedAfter)
}

func vmUninstallDiagnosis(code int, installedAfter bool, attempt, maxAttempts int) string {
	if code == uninstallCodeOK && installedAfter {
		return fmt.Sprintf("uninstaller returned success but target remains detected after attempt %d/%d", attempt, maxAttempts)
	}
	switch code {
	case uninstallCodeRunningProcess:
		return fmt.Sprintf("uninstall blocked by a running process on attempt %d/%d", attempt, maxAttempts)
	case uninstallCodeFailed:
		return fmt.Sprintf("all safe production uninstall paths failed and target remains installed on attempt %d/%d", attempt, maxAttempts)
	case uninstallCodeUnsupported:
		return "no safe automatic uninstall strategy is available"
	case uninstallCodeNeedElevation:
		return "machine-scope target requires elevation"
	case uninstallCodeWrongElevation:
		return "user-scope target cannot be safely removed from the elevated token"
	default:
		return fmt.Sprintf("uninstall returned code %d and installed_after=%t", code, installedAfter)
	}
}
