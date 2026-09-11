//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/quality"
)

type vmTestEvent struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type vmDetected struct {
	Found            bool   `json:"found"`
	Name             string `json:"name,omitempty"`
	ID               string `json:"id,omitempty"`
	Version          string `json:"version,omitempty"`
	Source           string `json:"source,omitempty"`
	Scope            string `json:"scope,omitempty"`
	InstallLocation  string `json:"install_location,omitempty"`
	RegistryKey      string `json:"registry_key,omitempty"`
	WindowsInstaller bool   `json:"windows_installer,omitempty"`
	Strategy         string `json:"uninstall_strategy,omitempty"`
}

type vmTestResult struct {
	SchemaVersion        int           `json:"schema_version"`
	AppVersion           string        `json:"app_version"`
	CatalogIndex         int           `json:"catalog_index"`
	Name                 string        `json:"name"`
	Category             string        `json:"category"`
	Profile              any           `json:"profile"`
	StartedAt            string        `json:"started_at"`
	FinishedAt           string        `json:"finished_at"`
	DurationSeconds      float64       `json:"duration_seconds"`
	VMID                 string        `json:"vm_id"`
	Executor             string        `json:"executor"`
	Precheck             string        `json:"precheck"`
	ResolvedID           string        `json:"resolved_id,omitempty"`
	ResolvedSource       string        `json:"resolved_source,omitempty"`
	ResolutionError      string        `json:"resolution_error,omitempty"`
	DownloadExitCode     int           `json:"download_exit_code,omitempty"`
	DownloadOK           bool          `json:"download_ok"`
	DownloadArtifact     bool          `json:"download_artifact_present"`
	InstallExitCode      int           `json:"install_exit_code,omitempty"`
	InstallOK            bool          `json:"install_ok"`
	InstallVerified      bool          `json:"install_verified"`
	DetectedAfterInstall vmDetected    `json:"detected_after_install"`
	UninstallAttempted   bool          `json:"uninstall_attempted"`
	UninstallExitCode    int           `json:"uninstall_exit_code,omitempty"`
	UninstallOK          bool          `json:"uninstall_ok"`
	UninstallVerified    bool          `json:"uninstall_verified"`
	FinalStatus          string        `json:"final_status"`
	RootCause            string        `json:"root_cause"`
	CoverageStatus       string        `json:"coverage_status"`
	SkipReason           string        `json:"skip_reason,omitempty"`
	FailureStage         string        `json:"failure_stage,omitempty"`
	Failure              string        `json:"failure,omitempty"`
	RebootRequired       bool          `json:"reboot_required"`
	Events               []vmTestEvent `json:"events"`
}

func vmTestRequested() bool {
	return len(os.Args) >= 4 && (os.Args[1] == "--vm-test-one" || os.Args[1] == "--vm-test-resume")
}

func runVMTestFromArgs() int {
	if os.Getenv("WIZ4RDFR0G_VM_TEST") != "1" || strings.TrimSpace(os.Getenv("WIZ4RDFR0G_VM_ID")) == "" {
		return 90
	}
	idx, err := strconvAtoiStrict(os.Args[2])
	if err != nil || idx < 0 || idx >= len(catalog) {
		return 91
	}
	resultPath, err := filepath.Abs(os.Args[3])
	if err != nil {
		return 92
	}
	workRoot := ""
	if len(os.Args) >= 5 {
		workRoot = os.Args[4]
	}
	if strings.TrimSpace(workRoot) == "" {
		workRoot = filepath.Join(os.TempDir(), "Wiz4rdFr0g-VMTest", fmt.Sprintf("%04d", idx))
	}
	var result vmTestResult
	if os.Args[1] == "--vm-test-resume" {
		result = resumeVMTestOne(idx, resultPath)
	} else {
		result = runVMTestOne(idx, workRoot)
	}
	if err := os.MkdirAll(filepath.Dir(resultPath), 0755); err != nil {
		return 93
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(resultPath, append(b, '\n'), 0644); err != nil {
		return 94
	}
	if result.FinalStatus == "FULL_PASS" || strings.HasPrefix(result.FinalStatus, "SKIPPED_") || result.FinalStatus == "SYSTEM_COMPONENT" || result.FinalStatus == "MANUAL_ONLY" || result.FinalStatus == "LICENSE_REQUIRED" || result.FinalStatus == "UNAVAILABLE" {
		return 0
	}
	return 1
}

func resumeVMTestOne(idx int, resultPath string) vmTestResult {
	app := catalog[idx]
	b, err := os.ReadFile(resultPath)
	if err != nil {
		return vmTestResult{SchemaVersion: 1, AppVersion: appVersion, CatalogIndex: idx, Name: app.Name, Category: app.Category, FinalStatus: "RESUME_FAIL", FailureStage: "RESUME", Failure: err.Error(), FinishedAt: time.Now().UTC().Format(time.RFC3339)}
	}
	var r vmTestResult
	if err := json.Unmarshal(b, &r); err != nil {
		return vmTestResult{SchemaVersion: 1, AppVersion: appVersion, CatalogIndex: idx, Name: app.Name, Category: app.Category, FinalStatus: "RESUME_FAIL", FailureStage: "RESUME", Failure: err.Error(), FinishedAt: time.Now().UTC().Format(time.RFC3339)}
	}
	if r.CatalogIndex != idx || r.Name != app.Name || r.FinalStatus != "SKIPPED_REBOOT_REQUIRED" {
		r.FinalStatus = "RESUME_FAIL"
		r.FailureStage = "RESUME"
		r.Failure = "checkpoint is not a matching reboot-required test result"
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return r
	}
	started := time.Now()
	if t, parseErr := time.Parse(time.RFC3339, r.StartedAt); parseErr == nil {
		started = t
	}
	log := func(level, msg string) {
		r.Events = append(r.Events, vmTestEvent{Time: time.Now().UTC().Format(time.RFC3339Nano), Level: level, Message: strings.TrimSpace(msg)})
	}
	finish := func(status string) vmTestResult {
		r.FinalStatus = status
		facts := quality.ResultFacts{FinalStatus: r.FinalStatus, FailureStage: r.FailureStage, Failure: r.Failure, SkipReason: r.SkipReason, InstallVerified: r.InstallVerified, UninstallVerified: r.UninstallVerified}
		r.RootCause = string(quality.ClassifyRootCause(facts))
		r.CoverageStatus = string(quality.CoverageFor(facts))
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		r.DurationSeconds = time.Since(started).Seconds()
		return r
	}
	log("INFO", "resumed physical test after Windows reboot")
	if !verifyProgramInstalled(app) {
		r.FailureStage = "INSTALL_VERIFY_AFTER_REBOOT"
		r.Failure = "program is not detected after the required reboot"
		return finish("VERIFY_FAIL")
	}
	r.InstallVerified = true
	detected, found := vmDetectInstalled(app)
	if !found {
		r.FailureStage = "DETECTION_AFTER_REBOOT"
		r.Failure = "production detector failed after reboot"
		return finish("DETECTION_FAIL")
	}
	r.DetectedAfterInstall = detected
	if !strategyAllowsAutomaticUninstall(catalogpkg.UninstallStrategy(detected.Strategy)) {
		r.SkipReason = "installed instance has no safe automatic uninstall strategy"
		return finish("MANUAL_ONLY")
	}
	r.UninstallAttempted = true
	uctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	scope := strings.ToLower(strings.TrimSpace(detected.Scope))
	var code int
	switch scope {
	case "user":
		if vmProcessElevated() {
			r.SkipReason = "user-scope uninstall must run with a standard user token; current executor is elevated"
			return finish("SKIPPED_PRIVILEGE_ENVIRONMENT")
		}
		code = uninstallInContext(uctx, app, false)
	case "machine":
		if !vmProcessElevated() {
			r.SkipReason = "machine-scope uninstall requires an elevated VM test process"
			return finish("SKIPPED_PRIVILEGE_ENVIRONMENT")
		}
		code = uninstallInContext(uctx, app, true)
	default:
		code = uninstallInContext(uctx, app, false)
		if code == 10 {
			if !vmProcessElevated() {
				r.SkipReason = "dynamic-scope package requires elevation but current executor is not elevated"
				return finish("SKIPPED_PRIVILEGE_ENVIRONMENT")
			}
			code = uninstallInContext(uctx, app, true)
		}
	}
	r.UninstallExitCode = code
	r.UninstallOK = code == 0
	if !r.UninstallOK {
		r.FailureStage = "UNINSTALL"
		r.Failure = fmt.Sprintf("production uninstall flow returned code %d", code)
		return finish("UNINSTALL_FAIL")
	}
	r.UninstallVerified = verifyProgramRemoved(app)
	if !r.UninstallVerified {
		r.FailureStage = "UNINSTALL_VERIFY"
		r.Failure = "uninstall returned success but production detector still finds the program"
		return finish("VERIFY_FAIL")
	}
	log("PASS", "post-reboot uninstall verified")
	return finish("FULL_PASS")
}

func runVMTestOne(idx int, workRoot string) vmTestResult {
	started := time.Now()
	app := catalog[idx]
	profile := catalogpkg.ProfileFor(app)
	r := vmTestResult{
		SchemaVersion:     1,
		AppVersion:        appVersion,
		CatalogIndex:      idx,
		Name:              app.Name,
		Category:          app.Category,
		Profile:           profile,
		StartedAt:         started.UTC().Format(time.RFC3339),
		VMID:              os.Getenv("WIZ4RDFR0G_VM_ID"),
		Executor:          vmExecutorName(),
		DownloadExitCode:  -999,
		InstallExitCode:   -999,
		UninstallExitCode: -999,
		FinalStatus:       "STARTED",
	}
	log := func(level, msg string) {
		r.Events = append(r.Events, vmTestEvent{Time: time.Now().UTC().Format(time.RFC3339Nano), Level: level, Message: strings.TrimSpace(msg)})
	}
	finish := func(status string) vmTestResult {
		r.FinalStatus = status
		facts := quality.ResultFacts{FinalStatus: r.FinalStatus, FailureStage: r.FailureStage, Failure: r.Failure, SkipReason: r.SkipReason, InstallVerified: r.InstallVerified, UninstallVerified: r.UninstallVerified}
		r.RootCause = string(quality.ClassifyRootCause(facts))
		r.CoverageStatus = string(quality.CoverageFor(facts))
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		r.DurationSeconds = time.Since(started).Seconds()
		return r
	}

	if profile.SystemComponent || profile.UninstallStrategy == catalogpkg.StrategySystemComponentUnsupported {
		r.Precheck = "SYSTEM_COMPONENT"
		r.SkipReason = "catalog profile marks this entry as a Windows-managed system component"
		log("SKIP", r.SkipReason)
		return finish("SYSTEM_COMPONENT")
	}
	if profile.UninstallStrategy == catalogpkg.StrategyManualOnly || profile.UninstallSupport == catalogpkg.SupportManual {
		r.Precheck = "MANUAL_ONLY"
		r.SkipReason = "catalog profile requires manual uninstall"
		log("SKIP", r.SkipReason)
		return finish("MANUAL_ONLY")
	}
	if _, err := exec.LookPath("winget.exe"); err != nil {
		r.Precheck = "NO_WINGET"
		r.SkipReason = "winget.exe is unavailable in this Windows executor"
		log("SKIP", r.SkipReason)
		return finish("UNAVAILABLE")
	}

	pre, found := vmDetectInstalled(app)
	if found {
		r.Precheck = "PREEXISTING"
		r.SkipReason = "program already existed before the test; harness refuses to remove pre-existing software"
		r.DetectedAfterInstall = pre
		log("SKIP", r.SkipReason)
		return finish("SKIPPED_PREEXISTING")
	}
	r.Precheck = "CLEAN"

	id, source, _, err := loadPackageData(app)
	if err != nil || id == "" || source == "" {
		r.ResolutionError = errorText(err)
		r.FailureStage = "PRECHECK"
		r.Failure = "no exact production package resolution"
		log("ERROR", r.Failure+": "+r.ResolutionError)
		return finish("UNAVAILABLE")
	}
	r.ResolvedID, r.ResolvedSource = id, source
	log("INFO", fmt.Sprintf("resolved exact package: %s (%s)", id, source))

	if source == "msstore" {
		r.SkipReason = "Microsoft Store/MSIX package requires a Store-capable interactive test image; generic CI install is not treated as physical PASS"
		log("SKIP", r.SkipReason)
		return finish("SKIPPED_STORE_ENVIRONMENT")
	}

	_ = os.RemoveAll(workRoot)
	if err := os.MkdirAll(workRoot, 0755); err != nil {
		r.FailureStage = "PRECHECK"
		r.Failure = err.Error()
		return finish("PRECHECK_FAIL")
	}
	defer os.RemoveAll(workRoot)
	downloadDir := filepath.Join(workRoot, "download")
	_ = os.MkdirAll(downloadDir, 0755)

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()

	dargs := []string{"download", "--id", id, "--exact", "--source", source, "--download-directory", downloadDir, "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
	log("CMD", formatCommand("winget.exe", dargs))
	dcode, dout, derr := runDirectProcess(ctx, "winget.exe", dargs)
	r.DownloadExitCode = dcode
	r.DownloadArtifact = downloadedArtifactExists(downloadDir)
	r.DownloadOK = derr == nil && dcode == 0 && r.DownloadArtifact
	if !r.DownloadOK {
		log("WARN", fmt.Sprintf("download did not produce a verified installer: exit=%d err=%s out=%s", dcode, errorText(derr), compactLog(dout)))
	} else {
		log("PASS", "installer artifact downloaded and verified non-empty")
	}

	iargs := []string{"install", "--id", id, "--exact", "--source", source, "--silent", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
	log("CMD", formatCommand("winget.exe", iargs))
	icode, iout, ierr := runDirectProcess(ctx, "winget.exe", iargs)
	r.InstallExitCode = icode
	if icode == 1641 || icode == 3010 {
		r.RebootRequired = true
	}
	r.InstallOK = ierr == nil && (icode == 0 || icode == 1641 || icode == 3010)
	if !r.InstallOK {
		r.FailureStage = "INSTALL"
		r.Failure = fmt.Sprintf("winget install failed: exit=%d err=%s out=%s", icode, errorText(ierr), compactLog(iout))
		log("ERROR", r.Failure)
		return finish("INSTALL_FAIL")
	}
	if r.RebootRequired {
		r.SkipReason = "installer reported a required Windows restart; physical PASS is blocked until the test is resumed after reboot"
		log("SKIP", r.SkipReason)
		return finish("SKIPPED_REBOOT_REQUIRED")
	}

	r.InstallVerified = verifyProgramInstalled(app)
	if !r.InstallVerified {
		r.FailureStage = "INSTALL_VERIFY"
		r.Failure = "installer returned success but production detector could not verify installed state"
		log("ERROR", r.Failure)
		return finish("VERIFY_FAIL")
	}
	log("PASS", "installation verified by production detector")

	detected, found := vmDetectInstalled(app)
	if !found {
		r.FailureStage = "DETECTION"
		r.Failure = "production detector failed after installation"
		log("ERROR", r.Failure)
		return finish("DETECTION_FAIL")
	}
	r.DetectedAfterInstall = detected
	log("PASS", fmt.Sprintf("detected name=%q id=%q version=%q scope=%q strategy=%q", detected.Name, detected.ID, detected.Version, detected.Scope, detected.Strategy))

	if !strategyAllowsAutomaticUninstall(catalogpkg.UninstallStrategy(detected.Strategy)) {
		r.SkipReason = "installed instance has no safe automatic uninstall strategy"
		log("SKIP", r.SkipReason)
		return finish("MANUAL_ONLY")
	}

	r.UninstallAttempted = true
	uctx, ucancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer ucancel()
	scope := strings.ToLower(strings.TrimSpace(detected.Scope))
	var ucode int
	switch scope {
	case "user":
		if vmProcessElevated() {
			r.SkipReason = "user-scope uninstall must run with a standard user token; current executor is elevated"
			log("SKIP", r.SkipReason)
			return finish("SKIPPED_PRIVILEGE_ENVIRONMENT")
		}
		ucode = uninstallInContext(uctx, app, false)
	case "machine":
		if !vmProcessElevated() {
			r.SkipReason = "machine-scope uninstall requires an elevated VM test process"
			log("SKIP", r.SkipReason)
			return finish("SKIPPED_PRIVILEGE_ENVIRONMENT")
		}
		ucode = uninstallInContext(uctx, app, true)
	default:
		ucode = uninstallInContext(uctx, app, false)
		if ucode == 10 {
			if !vmProcessElevated() {
				r.SkipReason = "dynamic-scope package requires elevation but current executor is not elevated"
				log("SKIP", r.SkipReason)
				return finish("SKIPPED_PRIVILEGE_ENVIRONMENT")
			}
			ucode = uninstallInContext(uctx, app, true)
		}
	}
	r.UninstallExitCode = ucode
	r.UninstallOK = ucode == 0
	if !r.UninstallOK {
		r.FailureStage = "UNINSTALL"
		r.Failure = fmt.Sprintf("production uninstall flow returned code %d", ucode)
		log("ERROR", r.Failure)
		return finish("UNINSTALL_FAIL")
	}

	r.UninstallVerified = verifyProgramRemoved(app)
	if !r.UninstallVerified {
		r.FailureStage = "UNINSTALL_VERIFY"
		r.Failure = "uninstall returned success but production detector still finds the program"
		log("ERROR", r.Failure)
		return finish("VERIFY_FAIL")
	}
	log("PASS", "uninstall verified; production detector no longer finds the program")
	return finish("FULL_PASS")
}

func vmDetectInstalled(app appDef) (vmDetected, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	regs := scanRegistryPackages()
	userRegs := registryPackagesByScope(regs, "user")
	machineRegs := registryPackagesByScope(regs, "machine")
	userPkgs := listInstalledPackagesForScope(ctx, "user")
	machinePkgs := listInstalledPackagesForScope(ctx, "machine")
	if pkg, ok := resolveInstalledPackage(app, userPkgs, userRegs); ok {
		return vmDetectedFrom(app, pkg, "user"), true
	}
	if pkg, ok := resolveInstalledPackage(app, machinePkgs, machineRegs); ok {
		return vmDetectedFrom(app, pkg, "machine"), true
	}
	allPkgs := append(append([]installedPackage(nil), userPkgs...), machinePkgs...)
	if pkg, ok := resolveInstalledPackage(app, allPkgs, regs); ok {
		return vmDetectedFrom(app, pkg, pkg.Scope), true
	}
	return vmDetected{}, false
}

func vmDetectedFrom(app appDef, pkg installedPackage, scope string) vmDetected {
	if pkg.Scope != "" {
		scope = pkg.Scope
	}
	strategy := installedUninstallStrategy(app, pkg)
	return vmDetected{
		Found:            true,
		Name:             pkg.Name,
		ID:               pkg.ID,
		Version:          pkg.Version,
		Source:           pkg.Source,
		Scope:            scope,
		InstallLocation:  pkg.InstallLocation,
		RegistryKey:      pkg.RegistryKey,
		WindowsInstaller: pkg.WindowsInstaller,
		Strategy:         string(strategy),
	}
}

func vmProcessElevated() bool {
	for _, candidate := range [][]string{{"fltmc.exe"}, {"net.exe", "session"}} {
		cmd := exec.Command(candidate[0], candidate[1:]...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if cmd.Run() == nil {
			return true
		}
	}
	return false
}

func vmExecutorName() string {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return "github-actions/windows"
	}
	if os.Getenv("TF_BUILD") == "True" {
		return "azure-pipelines/windows"
	}
	return "windows-vm"
}

func compactLog(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " | "))
	if len(s) > 1200 {
		return s[:1200] + "..."
	}
	return s
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func strconvAtoiStrict(s string) (int, error) {
	n := 0
	if strings.TrimSpace(s) == "" {
		return 0, fmt.Errorf("empty integer")
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid integer")
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}
