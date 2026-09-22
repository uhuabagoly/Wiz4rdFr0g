//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type vmUninstallAttemptRecord struct {
	Attempt         int        `json:"attempt"`
	Scope           string     `json:"scope,omitempty"`
	Strategy        string     `json:"strategy,omitempty"`
	Elevated        bool       `json:"elevated"`
	ExitCode        int        `json:"exit_code"`
	InstalledAfter  bool       `json:"installed_after"`
	VerifiedRemoval bool       `json:"verified_removal"`
	RepairAction    string     `json:"repair_action,omitempty"`
	Diagnosis       string     `json:"diagnosis,omitempty"`
	WorkerLog       string     `json:"worker_log,omitempty"`
	DetectedBefore  vmDetected `json:"detected_before"`
	DetectedAfter   vmDetected `json:"detected_after,omitempty"`
}

type vmUninstallOutcome struct {
	Code             int
	Success          bool
	VerifiedRemoval  bool
	RebootRequired   bool
	PrivilegeBlocked bool
	SkipReason       string
	Diagnosis        string
	Attempts         []vmUninstallAttemptRecord
}

func vmExecuteUninstallWithRepair(parent context.Context, app appDef, initial vmDetected, log func(string, string)) vmUninstallOutcome {
	maxAttempts := 1 + campaignRetryLimitFromEnv("WIZ4RDFR0G_UNINSTALL_RETRIES", 1)
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	current := initial
	outcome := vmUninstallOutcome{Code: uninstallCodeFailed}
	for attemptNo := 1; attemptNo <= maxAttempts; attemptNo++ {
		attemptTimeout := campaignDurationFromEnv("WIZ4RDFR0G_UNINSTALL_TIMEOUT_MINUTES", 25*time.Minute, 3*time.Minute, 90*time.Minute)
		ctx, cancel := context.WithTimeout(parent, attemptTimeout)
		logOffset := currentLogSize()
		code, elevated, skipReason := vmUninstallOnce(ctx, app, current)
		cancel()

		record := vmUninstallAttemptRecord{
			Attempt: attemptNo, Scope: current.Scope, Strategy: current.Strategy, Elevated: elevated,
			ExitCode: code, DetectedBefore: current,
		}
		record.WorkerLog = compactLog(readLogSince(logOffset, 12000))
		if skipReason != "" {
			record.Diagnosis = skipReason
			outcome.Code = code
			outcome.PrivilegeBlocked = code == uninstallCodeNeedElevation || code == uninstallCodeWrongElevation
			outcome.SkipReason = skipReason
			outcome.Diagnosis = skipReason
			outcome.Attempts = append(outcome.Attempts, record)
			return outcome
		}
		if code == uninstallCodeRebootRequired {
			record.Diagnosis = "uninstaller reported reboot required"
			outcome.Code = code
			outcome.RebootRequired = true
			outcome.Diagnosis = record.Diagnosis
			outcome.Attempts = append(outcome.Attempts, record)
			return outcome
		}

		if code == uninstallCodeOK {
			removed := verifyProgramRemoved(app)
			record.VerifiedRemoval = removed
			record.InstalledAfter = !removed
			if removed {
				record.Diagnosis = "uninstall command succeeded and independent post-state verification confirmed absence"
				outcome.Code = uninstallCodeOK
				outcome.Success = true
				outcome.VerifiedRemoval = true
				outcome.Diagnosis = record.Diagnosis
				outcome.Attempts = append(outcome.Attempts, record)
				return outcome
			}
		} else {
			after, state := vmDetectInstalledState(app)
			if state == vmDetectionAmbiguous {
				record.InstalledAfter = true
				record.Diagnosis = "post-uninstall detection became ambiguous; absence cannot be proven"
				outcome.Code = code
				outcome.Diagnosis = record.Diagnosis
				outcome.Attempts = append(outcome.Attempts, record)
				return outcome
			}
			still := state == vmDetectionFound
			record.InstalledAfter = still
			record.VerifiedRemoval = state == vmDetectionAbsent
			if still {
				record.DetectedAfter = after
			} else {
				record.Diagnosis = fmt.Sprintf("uninstaller returned code %d although independent detection no longer finds the target; non-zero command status prevents FULL_PASS", code)
				outcome.Code = code
				outcome.VerifiedRemoval = true
				outcome.Diagnosis = record.Diagnosis
				outcome.Attempts = append(outcome.Attempts, record)
				return outcome
			}
		}

		if !record.InstalledAfter {
			record.InstalledAfter = true
		}

		if !shouldRetryUninstall(code, record.InstalledAfter, attemptNo, maxAttempts) {
			record.Diagnosis = vmUninstallDiagnosis(code, record.InstalledAfter, attemptNo, maxAttempts)
			outcome.Code = code
			outcome.Diagnosis = record.Diagnosis
			outcome.Attempts = append(outcome.Attempts, record)
			return outcome
		}

		repairAction := "redetect target and retry the production uninstall flow"
		if code == uninstallCodeRunningProcess {
			closed, action := vmCloseDetectedProcesses(current)
			if closed {
				repairAction = action
			} else if action != "" {
				repairAction = action + "; then redetect and retry"
			}
		}
		after, state := vmDetectInstalledState(app)
		if state == vmDetectionAmbiguous {
			record.RepairAction = repairAction
			record.Diagnosis = "repair redetection is ambiguous; further destructive retries refused"
			outcome.Code = code
			outcome.Diagnosis = record.Diagnosis
			outcome.Attempts = append(outcome.Attempts, record)
			return outcome
		}
		if state == vmDetectionFound {
			if !strings.EqualFold(after.Strategy, current.Strategy) || !strings.EqualFold(after.Scope, current.Scope) || !strings.EqualFold(after.ID, current.ID) {
				repairAction += fmt.Sprintf("; runtime identity changed to id=%q scope=%q strategy=%q", after.ID, after.Scope, after.Strategy)
			}
			current = after
		}
		record.RepairAction = repairAction
		record.Diagnosis = vmUninstallDiagnosis(code, true, attemptNo, maxAttempts)
		outcome.Attempts = append(outcome.Attempts, record)
		log("REPAIR", fmt.Sprintf("%s: uninstall attempt %d failed (code=%d, strategy=%s, scope=%s); %s", app.Name, attemptNo, code, record.Strategy, record.Scope, repairAction))
		time.Sleep(750 * time.Millisecond)
	}
	outcome.Diagnosis = "repair attempts exhausted"
	return outcome
}

func vmUninstallOnce(ctx context.Context, app appDef, detected vmDetected) (code int, elevated bool, skipReason string) {
	scope := strings.ToLower(strings.TrimSpace(detected.Scope))
	elevated = vmProcessElevated()
	switch scope {
	case "user":
		if elevated {
			self, err := os.Executable()
			if err != nil {
				return uninstallCodeWrongElevation, true, err.Error()
			}
			for index, candidate := range catalog {
				if candidate.Name != app.Name {
					continue
				}
				code, output, err := runStandardUserProcess(ctx, self, []string{"--uninstall-worker-user", strconv.Itoa(index)})
				if code < 0 {
					return uninstallCodeWrongElevation, true, fmt.Sprintf("standard-user worker could not start: %v %s", err, compactLog(output))
				}
				return code, false, ""
			}
			return uninstallCodeUnsupported, true, "exact catalog index for user worker not found"
		}
		return uninstallInContext(ctx, app, false), false, ""
	case "machine":
		if !elevated {
			return uninstallCodeNeedElevation, false, "machine-scope uninstall requires an elevated test executor"
		}
		return uninstallInContext(ctx, app, true), true, ""
	default:
		return uninstallInContext(ctx, app, elevated), elevated, ""
	}
}

func vmCloseDetectedProcesses(d vmDetected) (bool, string) {
	root := filepath.Clean(strings.TrimSpace(d.InstallLocation))
	if root == "" || root == "." || !filepath.IsAbs(root) {
		return false, "no trustworthy absolute install root is available for automatic process closure"
	}
	lower := strings.ToLower(root)
	if strings.Contains(lower, `\windows\`) || strings.HasSuffix(lower, `\windows`) {
		return false, "install root points into the Windows directory; automatic process closure refused"
	}
	closed, needElevation, err := terminateKnownProcesses([]string{root}, nil)
	if err != nil {
		return false, "safe process closure failed: " + err.Error()
	}
	if needElevation {
		return false, "matching processes require a different privilege context"
	}
	return true, fmt.Sprintf("closed %d process(es) whose executable path is under the verified install root %q", closed, root)
}

type vmCommandRun struct {
	ExitCode int
	Output   string
	Err      error
	Retries  int
}

func vmRunCommandWithTransientRetry(ctx context.Context, exe string, args []string, maxRetries int, canRetry func() bool, log func(string, string)) vmCommandRun {
	var result vmCommandRun
	for attempt := 0; ; attempt++ {
		code, out, err := runDirectProcess(ctx, exe, args)
		result.ExitCode, result.Output, result.Err = code, out, err
		if err == nil {
			return result
		}
		if ctx.Err() != nil || attempt >= maxRetries || !isTransientPackageManagerFailure(out+" "+err.Error()) {
			return result
		}
		if canRetry != nil && !canRetry() {
			return result
		}
		result.Retries++
		if log != nil {
			log("RETRY", fmt.Sprintf("transient package-manager failure; retry %d/%d: exit=%d %s", result.Retries, maxRetries, code, compactLog(out)))
		}
		select {
		case <-ctx.Done():
			result.Err = ctx.Err()
			return result
		case <-time.After(2 * time.Second):
		}
	}
}
