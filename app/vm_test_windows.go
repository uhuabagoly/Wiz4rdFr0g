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
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
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
	CampaignID           string                     `json:"campaign_id"`
	CampaignAttempt      string                     `json:"attempt"`
	DownloadProof        vmDownloadProof            `json:"download_proof"`
	FilesystemProof      vmFilesystemProof          `json:"filesystem_proof"`
	Phase                string                     `json:"phase"`
	Environment          json.RawMessage            `json:"environment"`
	IndependentBefore    vmIndependentState         `json:"independent_before"`
	IndependentInstalled vmIndependentState         `json:"independent_installed"`
	IndependentRemoved   vmIndependentState         `json:"independent_removed"`
	DocumentSignature    string                     `json:"document_signature"`
	SchemaVersion        int                        `json:"schema_version"`
	BuildID              string                     `json:"build_id"`
	AppVersion           string                     `json:"app_version"`
	GitCommit            string                     `json:"git_commit"`
	CatalogFingerprint   string                     `json:"catalog_fingerprint"`
	ArtifactSHA256       string                     `json:"artifact_sha256"`
	TestRunID            string                     `json:"test_run_id"`
	CatalogIndex         int                        `json:"catalog_index"`
	CatalogAppID         string                     `json:"catalog_app_id"`
	CatalogAppName       string                     `json:"catalog_app_name"`
	Name                 string                     `json:"name"`
	Category             string                     `json:"category"`
	Profile              any                        `json:"profile"`
	StartedAt            string                     `json:"started_at"`
	FinishedAt           string                     `json:"finished_at"`
	DurationSeconds      float64                    `json:"duration_seconds"`
	VMID                 string                     `json:"vm_id"`
	MachineID            string                     `json:"machine_id"`
	Executor             string                     `json:"executor"`
	Precheck             string                     `json:"precheck"`
	ResolvedID           string                     `json:"resolved_id,omitempty"`
	ResolvedSource       string                     `json:"resolved_source,omitempty"`
	ResolvedVersion      string                     `json:"resolved_version,omitempty"`
	ResolutionError      string                     `json:"resolution_error,omitempty"`
	DownloadExitCode     int                        `json:"download_exit_code"`
	DownloadRetryCount   int                        `json:"download_retry_count"`
	DownloadOK           bool                       `json:"download_ok"`
	DownloadArtifact     bool                       `json:"download_artifact_present"`
	InstallExitCode      int                        `json:"install_exit_code"`
	InstallRetryCount    int                        `json:"install_retry_count"`
	InstallOK            bool                       `json:"install_ok"`
	InstallVerified      bool                       `json:"install_verified"`
	DetectedAfterInstall vmDetected                 `json:"detected_after_install"`
	UninstallAttempted   bool                       `json:"uninstall_attempted"`
	UninstallExitCode    int                        `json:"uninstall_exit_code"`
	UninstallOK          bool                       `json:"uninstall_ok"`
	UninstallVerified    bool                       `json:"uninstall_verified"`
	UninstallDiagnosis   string                     `json:"uninstall_diagnosis,omitempty"`
	UninstallAttempts    []vmUninstallAttemptRecord `json:"uninstall_attempts,omitempty"`
	FinalStatus          string                     `json:"final_status"`
	RootCause            string                     `json:"root_cause"`
	CoverageStatus       string                     `json:"coverage_status"`
	SkipReason           string                     `json:"skip_reason,omitempty"`
	FailureStage         string                     `json:"failure_stage,omitempty"`
	Failure              string                     `json:"failure,omitempty"`
	RebootRequired       bool                       `json:"reboot_required"`
	Events               []vmTestEvent              `json:"events"`
	Signature            string                     `json:"signature"`
}

func (r vmTestResult) evidenceStatement() releaseproof.EvidenceStatement {
	return releaseproof.EvidenceStatement{
		SchemaVersion: r.SchemaVersion, BuildID: r.BuildID, AppVersion: r.AppVersion, GitCommit: r.GitCommit,
		CatalogFingerprint: r.CatalogFingerprint, ArtifactSHA256: r.ArtifactSHA256, TestRunID: r.TestRunID,
		CatalogIndex: r.CatalogIndex, CatalogAppID: r.CatalogAppID, CatalogAppName: r.CatalogAppName,
		StartedAt: r.StartedAt, FinishedAt: r.FinishedAt, MachineID: r.MachineID, FinalStatus: r.FinalStatus,
		FailureStage: r.FailureStage, Failure: r.Failure, SkipReason: r.SkipReason,
		InstallVerified: r.InstallVerified, UninstallVerified: r.UninstallVerified, DurationSeconds: r.DurationSeconds,
	}
}

func runtimeBuildManifest() (releaseproof.BuildManifest, error) {
	if !releaseproof.ValidGitCommit(buildGitCommit) {
		return releaseproof.BuildManifest{}, fmt.Errorf("build git commit is not embedded")
	}
	fingerprint, err := releaseproof.CurrentCatalogFingerprint()
	if err != nil {
		return releaseproof.BuildManifest{}, fmt.Errorf("catalog fingerprint: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return releaseproof.BuildManifest{}, fmt.Errorf("executable path: %w", err)
	}
	hash, size, err := releaseproof.FileSHA256(exe)
	if err != nil {
		return releaseproof.BuildManifest{}, fmt.Errorf("executable sha256: %w", err)
	}
	m := releaseproof.BuildManifest{
		SchemaVersion: releaseproof.BuildManifestSchemaVersion, AppVersion: appVersion,
		GitCommit: buildGitCommit, CatalogFingerprint: fingerprint, EvidenceSchemaVersion: releaseproof.EvidenceSchemaVersion,
		Artifacts:         map[string]releaseproof.Artifact{releaseproof.PrimaryWindowsArtifactKey: {Path: exe, SHA256: hash, Size: size}},
		PayloadConsistent: true,
	}
	m.BuildID = releaseproof.BuildID(m.AppVersion, m.GitCommit, m.CatalogFingerprint, hash, m.EvidenceSchemaVersion)
	return m, nil
}

func attachPhysicalEvidence(r *vmTestResult) error {
	if r == nil || r.CatalogIndex < 0 || r.CatalogIndex >= len(catalog) {
		return fmt.Errorf("invalid catalog index for evidence")
	}
	manifest, err := runtimeBuildManifest()
	if err != nil {
		return err
	}
	entries := catalogpkg.BuildAuditEntries()
	entry := entries[r.CatalogIndex]
	r.SchemaVersion = releaseproof.EvidenceSchemaVersion
	r.BuildID = manifest.BuildID
	r.AppVersion = manifest.AppVersion
	r.GitCommit = manifest.GitCommit
	r.CatalogFingerprint = manifest.CatalogFingerprint
	r.ArtifactSHA256 = manifest.Artifacts[releaseproof.PrimaryWindowsArtifactKey].SHA256
	r.TestRunID = strings.TrimSpace(os.Getenv("WIZ4RDFR0G_TEST_RUN_ID"))
	r.CatalogAppID = releaseproof.CatalogAppID(entry)
	r.CatalogAppName = entry.Name
	r.Name = entry.Name
	r.MachineID = strings.TrimSpace(os.Getenv("WIZ4RDFR0G_VM_ID"))
	r.VMID = r.MachineID
	if strings.TrimSpace(r.StartedAt) == "" {
		r.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if strings.TrimSpace(r.FinishedAt) == "" {
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	key := []byte(os.Getenv(releaseproof.EvidenceKeyEnvironment))
	sig, err := releaseproof.EvidenceSignature(r.evidenceStatement(), key)
	if err != nil {
		return err
	}
	r.Signature = sig
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	r.DocumentSignature, err = releaseproof.DocumentSignature(b, key)
	if err != nil {
		return err
	}
	return nil
}

func validateResumeCheckpoint(r vmTestResult, idx int) error {
	manifest, err := runtimeBuildManifest()
	if err != nil {
		return err
	}
	entries := catalogpkg.BuildAuditEntries()
	if idx < 0 || idx >= len(entries) {
		return fmt.Errorf("invalid checkpoint catalog index")
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if err := releaseproof.VerifyDocument(b, []byte(os.Getenv(releaseproof.EvidenceKeyEnvironment))); err != nil {
		return err
	}
	if r.MachineID != os.Getenv("WIZ4RDFR0G_VM_ID") || r.TestRunID != os.Getenv("WIZ4RDFR0G_TEST_RUN_ID") {
		return fmt.Errorf("checkpoint belongs to another VM/run; start a clean test")
	}
	issues := releaseproof.ValidateEvidence(r.evidenceStatement(), r.Signature, manifest, entries[idx], time.Now().UTC(), []byte(os.Getenv(releaseproof.EvidenceKeyEnvironment)))
	if len(issues) != 0 {
		parts := make([]string, 0, len(issues))
		for _, issue := range issues {
			parts = append(parts, issue.Code+": "+issue.Message)
		}
		return fmt.Errorf("checkpoint evidence validation failed: %s", strings.Join(parts, "; "))
	}
	return nil
}

func vmTestRequested() bool {
	return len(os.Args) >= 4 && (os.Args[1] == "--vm-test-one" || os.Args[1] == "--vm-test-resume")
}

func runVMTestFromArgs() int {
	if os.Getenv("WIZ4RDFR0G_VM_TEST") != "1" || strings.TrimSpace(os.Getenv("WIZ4RDFR0G_VM_ID")) == "" {
		return 90
	}
	if strings.TrimSpace(os.Getenv("WIZ4RDFR0G_TEST_RUN_ID")) == "" {
		return 95
	}
	if err := releaseproof.ValidateEvidenceKey([]byte(os.Getenv(releaseproof.EvidenceKeyEnvironment))); err != nil {
		return 96
	}
	if !releaseproof.ValidGitCommit(buildGitCommit) {
		return 97
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
	environment, preflightErr := vmActionsEnvironment(idx)
	if preflightErr != nil {
		result = vmTestResult{CatalogIndex: idx, FinalStatus: "BLOCKED_ENVIRONMENT", FailureStage: "ENVIRONMENT", Failure: preflightErr.Error()}
	} else if os.Args[1] == "--vm-test-resume" {
		result = resumeVMTestOne(idx, resultPath)
	} else {
		result = runVMTestOne(idx, workRoot, resultPath)
	}
	result.CampaignID = os.Getenv("WIZ4RDFR0G_CAMPAIGN_ID")
	result.CampaignAttempt = os.Getenv("WIZ4RDFR0G_CAMPAIGN_ATTEMPT")
	result.Environment = environment
	if err := os.MkdirAll(filepath.Dir(resultPath), 0755); err != nil {
		return 93
	}
	if err := attachPhysicalEvidence(&result); err != nil {
		result.FinalStatus = "EVIDENCE_FAIL"
		result.FailureStage = "EVIDENCE"
		result.Failure = err.Error()
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		// Write the failed result for diagnostics; an unsigned/invalid result can never pass the release gate.
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	if err := vmAtomicWrite(resultPath, append(b, '\n')); err != nil {
		return 94
	}
	if result.FinalStatus == "EVIDENCE_FAIL" {
		return 98
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
		return vmTestResult{SchemaVersion: releaseproof.EvidenceSchemaVersion, AppVersion: appVersion, CatalogIndex: idx, Name: app.Name, Category: app.Category, FinalStatus: "RESUME_FAIL", FailureStage: "RESUME", Failure: err.Error(), FinishedAt: time.Now().UTC().Format(time.RFC3339)}
	}
	var r vmTestResult
	if err := json.Unmarshal(b, &r); err != nil {
		return vmTestResult{SchemaVersion: releaseproof.EvidenceSchemaVersion, AppVersion: appVersion, CatalogIndex: idx, Name: app.Name, Category: app.Category, FinalStatus: "RESUME_FAIL", FailureStage: "RESUME", Failure: err.Error(), FinishedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	}
	if err := validateResumeCheckpoint(r, idx); err != nil {
		r.FinalStatus = "RESUME_FAIL"
		r.FailureStage = "RESUME_EVIDENCE"
		r.Failure = err.Error()
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return r
	}
	if os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" {
		r.FinalStatus = "REBOOT_PENDING"
		r.FailureStage = "RESUME"
		r.Failure = "GitHub-hosted jobs cannot restore a reboot checkpoint; rerun the entire lifecycle on a fresh job"
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return r
	}
	if r.CatalogIndex != idx || r.Name != app.Name || r.FinalStatus != "SKIPPED_REBOOT_REQUIRED" {
		r.FinalStatus = "RESUME_FAIL"
		r.FailureStage = "RESUME"
		r.Failure = "checkpoint is not a matching reboot-required test result"
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return r
	}
	// A hosted job cannot restore another job's machine. Reboots after uninstall
	// must never be mistaken for an install-stage checkpoint.
	if r.UninstallAttempted {
		r.IndependentRemoved = vmIndependentProbe(r.ResolvedID, r.ResolvedSource)
		_, state := vmDetectInstalledState(app)
		r.UninstallVerified = state == vmDetectionAbsent && r.IndependentRemoved.State == "absent"
		r.FinalStatus = "REBOOT_PENDING"
		r.Failure = "uninstall reboot requires a supported same-VM continuation; fresh Actions jobs must rerun the full lifecycle"
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
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
	r.IndependentInstalled = vmIndependentProbe(r.ResolvedID, r.ResolvedSource)
	if r.IndependentInstalled.State != "present" {
		return finish("INSTALL_VERIFY_FAIL")
	}
	detected, detectState := vmDetectInstalledState(app)
	if detectState == vmDetectionAmbiguous {
		r.FailureStage = "DETECTION_AFTER_REBOOT"
		r.Failure = "post-reboot detection is ambiguous; destructive uninstall is refused"
		return finish("AMBIGUOUS")
	}
	if detectState != vmDetectionFound {
		r.FailureStage = "DETECTION_AFTER_REBOOT"
		r.Failure = "production detector failed after reboot"
		return finish("DETECTION_FAIL")
	}
	probeCtx, probeCancel := context.WithTimeout(context.Background(), time.Minute)
	defer probeCancel()
	r.FilesystemProof, err = vmCaptureFilesystem(probeCtx, detected.Name)
	if err != nil {
		r.FailureStage = "INDEPENDENT_FILESYSTEM"
		r.Failure = err.Error()
		return finish("INSTALL_VERIFY_FAIL")
	}
	r.DetectedAfterInstall = detected
	if !strategyAllowsAutomaticUninstall(catalogpkg.UninstallStrategy(detected.Strategy)) {
		r.SkipReason = "installed instance has no safe automatic uninstall strategy"
		return finish("MANUAL_ONLY")
	}
	r.UninstallAttempted = true
	uctx, cancel := context.WithTimeout(context.Background(), campaignDurationFromEnv("WIZ4RDFR0G_UNINSTALL_TOTAL_TIMEOUT_MINUTES", 55*time.Minute, 5*time.Minute, 180*time.Minute))
	defer cancel()
	uout := vmExecuteUninstallWithRepair(uctx, app, detected, log)
	r.UninstallExitCode = uout.Code
	r.UninstallOK = uout.Success
	r.UninstallVerified = uout.VerifiedRemoval
	r.UninstallDiagnosis = uout.Diagnosis
	r.UninstallAttempts = append(r.UninstallAttempts, uout.Attempts...)
	if uout.RebootRequired {
		r.RebootRequired = true
		r.SkipReason = "uninstall still requires another Windows restart"
		return finish("SKIPPED_REBOOT_REQUIRED")
	}
	if uout.PrivilegeBlocked {
		r.SkipReason = uout.SkipReason
		return finish("SKIPPED_PRIVILEGE_ENVIRONMENT")
	}
	if !uout.Success {
		r.FailureStage = "UNINSTALL"
		r.Failure = uout.Diagnosis
		if len(uout.Attempts) > 1 {
			return finish("UNINSTALL_REPAIR_FAILED")
		}
		return finish("UNINSTALL_FAIL")
	}
	log("PASS", fmt.Sprintf("post-reboot uninstall verified after %d attempt(s)", len(uout.Attempts)))
	r.IndependentRemoved = vmIndependentProbe(r.ResolvedID, r.ResolvedSource)
	_, finalState := vmDetectInstalledState(app)
	if r.IndependentRemoved.State != "absent" || finalState != vmDetectionAbsent {
		return finish("UNINSTALL_VERIFY_FAIL")
	}
	if err := vmVerifyFilesystemRemoved(uctx, &r.FilesystemProof); err != nil {
		r.UninstallVerified = false
		r.FailureStage = "UNINSTALL_VERIFY"
		r.Failure = err.Error()
		return finish("UNINSTALL_VERIFY_FAIL")
	}
	return finish("FULL_PASS")
}

func runVMTestOne(idx int, workRoot, resultPath string) vmTestResult {
	started := time.Now()
	app := catalog[idx]
	profile := catalogpkg.ProfileFor(app)
	r := vmTestResult{
		SchemaVersion:     releaseproof.EvidenceSchemaVersion,
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
	checkpoint := func(phase string) error {
		r.Phase = phase
		copy := r
		copy.FinalStatus = "IN_PROGRESS"
		copy.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		copy.DurationSeconds = time.Since(started).Seconds()
		var err error
		copy.Environment, err = vmActionsEnvironment(idx)
		if err != nil {
			return err
		}
		if err := attachPhysicalEvidence(&copy); err != nil {
			return err
		}
		b, err := json.MarshalIndent(copy, "", "  ")
		if err != nil {
			return err
		}
		return vmAtomicWrite(resultPath, b)
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
	if !profile.License.PolicyOK {
		r.Precheck = "LICENSE_POLICY"
		r.SkipReason = fmt.Sprintf("license policy blocks physical installation until official-source review passes: class=%s source=%q", profile.License.Class, profile.License.SourceURL)
		log("SKIP", r.SkipReason)
		return finish("LICENSE_REQUIRED")
	}
	if _, err := exec.LookPath("winget.exe"); err != nil {
		r.Precheck = "NO_WINGET"
		r.SkipReason = "winget.exe is unavailable in this Windows executor"
		log("SKIP", r.SkipReason)
		return finish("UNAVAILABLE")
	}

	pre, preState := vmDetectInstalledState(app)
	if preState == vmDetectionAmbiguous {
		r.Precheck = "AMBIGUOUS"
		r.FailureStage = "PRECHECK"
		r.Failure = "multiple equally valid installed identities were detected; destructive testing is refused"
		log("ERROR", r.Failure)
		return finish("AMBIGUOUS")
	}
	if preState == vmDetectionFound {
		r.Precheck = "PREEXISTING"
		r.SkipReason = "program already existed before the test; harness refuses to remove pre-existing software"
		r.DetectedAfterInstall = pre
		log("SKIP", r.SkipReason)
		return finish("SKIPPED_PREEXISTING")
	}
	r.Precheck = "CLEAN"

	id, source, versions, err := loadPackageData(app)
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
	r.IndependentBefore = vmIndependentProbe(id, source)
	if r.IndependentBefore.State == "present" {
		r.Precheck = "PREEXISTING"
		return finish("SKIPPED_PREEXISTING")
	}
	if r.IndependentBefore.State != "absent" {
		r.Failure = "independent pre-install query failed; clean state is not proven"
		return finish("BLOCKED_ENVIRONMENT")
	}
	if len(versions) == 0 {
		r.Failure = "production resolver returned no version to pin download and installation"
		return finish("PACKAGE_UNAVAILABLE")
	}
	r.ResolvedVersion = versions[0]
	// Never recursively delete a caller-supplied directory. Own a newly created
	// temporary child and clean up only that child.
	workRoot, err = os.MkdirTemp(os.TempDir(), "Wiz4rdFr0g-physical-")
	if err != nil {
		r.FailureStage = "PRECHECK"
		r.Failure = err.Error()
		return finish("PRECHECK_FAIL")
	}
	defer os.RemoveAll(workRoot)
	downloadDir := filepath.Join(workRoot, "download")
	_ = os.MkdirAll(downloadDir, 0755)

	ctx, cancel := context.WithTimeout(context.Background(), campaignDurationFromEnv("WIZ4RDFR0G_INSTALL_TIMEOUT_MINUTES", 35*time.Minute, 5*time.Minute, 120*time.Minute))
	defer cancel()

	dargs := []string{"download", "--id", id, "--exact", "--source", source, "--download-directory", downloadDir, "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
	dargs = append(dargs, "--version", r.ResolvedVersion)
	log("CMD", formatCommand("winget.exe", dargs))
	if err := checkpoint("DOWNLOAD_PENDING"); err != nil {
		r.Failure = err.Error()
		return finish("CHECKPOINT_FAIL")
	}
	drun := vmRunCommandWithTransientRetry(ctx, "winget.exe", dargs, campaignRetryLimitFromEnv("WIZ4RDFR0G_PACKAGE_RETRIES", 1), nil, log)
	dcode, dout, derr := drun.ExitCode, drun.Output, drun.Err
	r.DownloadExitCode = dcode
	r.DownloadRetryCount = drun.Retries
	r.DownloadArtifact = downloadedArtifactExists(downloadDir)
	r.DownloadOK = derr == nil && dcode == 0 && r.DownloadArtifact
	if !r.DownloadOK {
		log("WARN", fmt.Sprintf("download did not produce a verified installer: exit=%d err=%s out=%s", dcode, errorText(derr), compactLog(dout)))
		r.FailureStage = "DOWNLOAD"
		r.Failure = "download failed or installer artifact missing"
		return finish("DOWNLOAD_FAIL")
	} else {
		log("PASS", "installer artifact downloaded and verified non-empty")
	}

	var proofErr error
	r.DownloadProof, proofErr = vmVerifyHTTPDownload(ctx, id, source, r.ResolvedVersion, workRoot)
	if proofErr != nil {
		r.DownloadOK = false
		r.FailureStage = "DOWNLOAD_VALIDATION"
		r.Failure = proofErr.Error()
		return finish("DOWNLOAD_FAIL")
	}
	iargs := packageInstallArgs(id, source)
	iargs = append(iargs, "--version", r.ResolvedVersion)
	log("CMD", formatCommand("winget.exe", iargs))
	if err := checkpoint("INSTALL_PENDING"); err != nil {
		r.Failure = err.Error()
		return finish("CHECKPOINT_FAIL")
	}
	irun := vmRunCommandWithTransientRetry(ctx, "winget.exe", iargs, campaignRetryLimitFromEnv("WIZ4RDFR0G_PACKAGE_RETRIES", 1), func() bool { return !verifyProgramInstalled(app) }, log)
	icode, iout, ierr := irun.ExitCode, irun.Output, irun.Err
	r.InstallExitCode = icode
	r.InstallRetryCount = irun.Retries
	if icode == 1641 || icode == 3010 {
		r.RebootRequired = true
	}
	r.InstallOK = ctx.Err() == nil && (icode == 0 || icode == 1641 || icode == 3010)
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
	r.IndependentInstalled = vmIndependentProbe(id, source)
	if r.IndependentInstalled.State != "present" {
		r.FailureStage = "INSTALL_VERIFY"
		r.Failure = "independent exact-ID query did not confirm installation"
		return finish("INSTALL_VERIFY_FAIL")
	}

	detected, detectState := vmDetectInstalledState(app)
	if detectState == vmDetectionAmbiguous {
		r.FailureStage = "DETECTION"
		r.Failure = "post-install detection is ambiguous; multiple equally valid target identities exist"
		log("ERROR", r.Failure)
		return finish("AMBIGUOUS")
	}
	if detectState != vmDetectionFound {
		r.FailureStage = "DETECTION"
		r.Failure = "production detector failed after installation"
		log("ERROR", r.Failure)
		return finish("DETECTION_FAIL")
	}
	r.FilesystemProof, err = vmCaptureFilesystem(ctx, detected.Name)
	if err != nil {
		r.FailureStage = "INDEPENDENT_FILESYSTEM"
		r.Failure = err.Error()
		return finish("INSTALL_VERIFY_FAIL")
	}
	r.DetectedAfterInstall = detected
	log("PASS", fmt.Sprintf("detected name=%q id=%q version=%q scope=%q strategy=%q", detected.Name, detected.ID, detected.Version, detected.Scope, detected.Strategy))
	if err := checkpoint("INSTALL_VERIFIED"); err != nil {
		r.Failure = err.Error()
		return finish("CHECKPOINT_FAIL")
	}

	if !strategyAllowsAutomaticUninstall(catalogpkg.UninstallStrategy(detected.Strategy)) {
		r.SkipReason = "installed instance has no safe automatic uninstall strategy"
		log("SKIP", r.SkipReason)
		return finish("MANUAL_ONLY")
	}

	r.UninstallAttempted = true
	if err := checkpoint("UNINSTALL_PENDING"); err != nil {
		r.Failure = err.Error()
		return finish("CHECKPOINT_FAIL")
	}
	uctx, ucancel := context.WithTimeout(context.Background(), campaignDurationFromEnv("WIZ4RDFR0G_UNINSTALL_TOTAL_TIMEOUT_MINUTES", 55*time.Minute, 5*time.Minute, 180*time.Minute))
	defer ucancel()
	uout := vmExecuteUninstallWithRepair(uctx, app, detected, log)
	r.UninstallExitCode = uout.Code
	r.UninstallOK = uout.Success
	r.UninstallVerified = uout.VerifiedRemoval
	r.UninstallDiagnosis = uout.Diagnosis
	r.UninstallAttempts = append(r.UninstallAttempts, uout.Attempts...)
	if uout.RebootRequired {
		r.RebootRequired = true
		r.SkipReason = "uninstaller reported a required Windows restart; physical PASS is blocked until the test is resumed after reboot"
		log("SKIP", r.SkipReason)
		return finish("SKIPPED_REBOOT_REQUIRED")
	}
	if uout.PrivilegeBlocked {
		r.SkipReason = uout.SkipReason
		log("SKIP", r.SkipReason)
		return finish("SKIPPED_PRIVILEGE_ENVIRONMENT")
	}
	if !uout.Success {
		r.FailureStage = "UNINSTALL"
		r.Failure = uout.Diagnosis
		if len(uout.Attempts) > 1 {
			log("ERROR", "uninstall repair attempts exhausted: "+r.Failure)
			return finish("UNINSTALL_REPAIR_FAILED")
		}
		log("ERROR", r.Failure)
		return finish("UNINSTALL_FAIL")
	}
	log("PASS", fmt.Sprintf("uninstall verified after %d attempt(s); production detector no longer finds the program", len(uout.Attempts)))
	r.IndependentRemoved = vmIndependentProbe(id, source)
	_, finalState := vmDetectInstalledState(app)
	if r.IndependentRemoved.State != "absent" || finalState != vmDetectionAbsent {
		r.UninstallVerified = false
		r.FailureStage = "UNINSTALL_VERIFY"
		r.Failure = "both detectors must confirm absence"
		return finish("UNINSTALL_VERIFY_FAIL")
	}
	if err := vmVerifyFilesystemRemoved(uctx, &r.FilesystemProof); err != nil {
		r.UninstallVerified = false
		r.FailureStage = "UNINSTALL_VERIFY"
		r.Failure = err.Error()
		return finish("UNINSTALL_VERIFY_FAIL")
	}
	return finish("FULL_PASS")
}

type vmDetectionState string

const (
	vmDetectionAbsent    vmDetectionState = "absent"
	vmDetectionFound     vmDetectionState = "found"
	vmDetectionAmbiguous vmDetectionState = "ambiguous"
)

func vmDetectInstalledState(app appDef) (vmDetected, vmDetectionState) {
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	regs := scanRegistryPackages()
	userRegs := registryPackagesByScope(regs, "user")
	machineRegs := registryPackagesByScope(regs, "machine")
	userPkgs := listInstalledPackagesForScope(ctx, "user")
	machinePkgs := listInstalledPackagesForScope(ctx, "machine")
	upkg, ustate := resolveInstalledPackageDetailed(app, userPkgs, userRegs)
	mpkg, mstate := resolveInstalledPackageDetailed(app, machinePkgs, machineRegs)
	if ustate == installedResolveAmbiguous || mstate == installedResolveAmbiguous {
		return vmDetected{}, vmDetectionAmbiguous
	}
	if ustate == installedResolveFound && mstate == installedResolveFound {
		return vmDetected{}, vmDetectionAmbiguous
	}
	if ustate == installedResolveFound {
		return vmDetectedFrom(app, upkg, "user"), vmDetectionFound
	}
	if mstate == installedResolveFound {
		return vmDetectedFrom(app, mpkg, "machine"), vmDetectionFound
	}
	allPkgs := append(append([]installedPackage(nil), userPkgs...), machinePkgs...)
	pkg, state := resolveInstalledPackageDetailed(app, allPkgs, regs)
	if state == installedResolveAmbiguous {
		return vmDetected{}, vmDetectionAmbiguous
	}
	if state == installedResolveFound {
		return vmDetectedFrom(app, pkg, pkg.Scope), vmDetectionFound
	}
	return vmDetected{}, vmDetectionAbsent
}

func vmDetectInstalled(app appDef) (vmDetected, bool) {
	d, state := vmDetectInstalledState(app)
	return d, state == vmDetectionFound
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
