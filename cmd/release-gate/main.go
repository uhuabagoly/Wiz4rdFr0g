package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/quality"
	"wiz4rdfr0g.local/fullcatalog/internal/releasegate"
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

type coverageRow struct {
	Index          int                    `json:"index"`
	Name           string                 `json:"name"`
	Category       string                 `json:"category"`
	CatalogAppID   string                 `json:"catalog_app_id"`
	Coverage       quality.CoverageStatus `json:"coverage_status"`
	RootCause      quality.RootCause      `json:"root_cause"`
	PhysicalResult string                 `json:"physical_result,omitempty"`
	Evidence       string                 `json:"evidence"`
}

type rootCauseRow struct {
	Index        int               `json:"index"`
	Name         string            `json:"name"`
	FinalStatus  string            `json:"final_status"`
	FailureStage string            `json:"failure_stage,omitempty"`
	RootCause    quality.RootCause `json:"root_cause"`
	Evidence     string            `json:"evidence"`
}

type regressionRow struct {
	Index      int    `json:"index"`
	Name       string `json:"name"`
	Evidence   string `json:"evidence"`
	BuildState string `json:"build_state"`
}
type regressionCorpus struct {
	PhysicalVerified []regressionRow `json:"physical_verified"`
	KnownRegressions json.RawMessage `json:"known_regressions"`
}

type gateReport struct {
	GeneratedAt             string            `json:"generated_at"`
	BuildID                 string            `json:"build_id,omitempty"`
	AppVersion              string            `json:"app_version,omitempty"`
	GitCommit               string            `json:"git_commit,omitempty"`
	CatalogFingerprint      string            `json:"catalog_fingerprint,omitempty"`
	ArtifactHashes          map[string]string `json:"artifact_hashes,omitempty"`
	WindowsSigningStatus    string            `json:"windows_signing_status,omitempty"`
	CatalogTotal            int               `json:"catalog_total"`
	PhysicalTestRequired    int               `json:"physical_test_required"`
	CatalogAuditPass        bool              `json:"catalog_audit_pass"`
	UnsafeResolutionEntries int               `json:"unsafe_resolution_entries"`
	PhysicalResultFiles     int               `json:"physical_result_files"`
	ValidPhysicalResults    int               `json:"valid_physical_results"`
	InvalidEvidence         int               `json:"invalid_evidence"`
	MismatchedEvidence      int               `json:"mismatched_or_stale_evidence"`
	DuplicateEvidence       int               `json:"duplicate_evidence"`
	ConflictingEvidence     int               `json:"conflicting_evidence"`
	VerifiedInstall         int               `json:"verified_install"`
	VerifiedUninstall       int               `json:"verified_uninstall"`
	VerifiedFull            int               `json:"verified_full"`
	VerifiedInstallOnly     int               `json:"verified_install_only"`
	FailedPhysical          int               `json:"failed_physical"`
	ManualUninstall         int               `json:"manual_uninstall"`
	SystemComponent         int               `json:"system_component"`
	LicenseBlocked          int               `json:"license_blocked"`
	Unavailable             int               `json:"unavailable"`
	Unresolved              int               `json:"unresolved"`
	MissingPhysicalEvidence int               `json:"missing_physical_evidence"`
	RootCauseCounts         map[string]int    `json:"root_cause_counts"`
	ReleaseReady            bool              `json:"release_ready"`
	Blockers                []string          `json:"blockers"`
}

func main() {
	resultsDir := "test/windows-vm/results"
	outputDir := "release"
	manifestPath := "release/build_manifest.json"
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		resultsDir = os.Args[1]
	}
	if len(os.Args) > 2 && strings.TrimSpace(os.Args[2]) != "" {
		outputDir = os.Args[2]
	}
	if len(os.Args) > 3 && strings.TrimSpace(os.Args[3]) != "" {
		manifestPath = os.Args[3]
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	auditEntries := catalogpkg.BuildAuditEntries()
	auditSummary := catalogpkg.Summarize(auditEntries)
	fingerprint, fpErr := releaseproof.CatalogFingerprint(auditEntries)

	report := gateReport{GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), CatalogTotal: len(auditEntries), PhysicalTestRequired: auditSummary.PhysicalTestRequired, CatalogAuditPass: auditSummary.Pass, UnsafeResolutionEntries: auditSummary.UnsafeResolutionEntries, RootCauseCounts: map[string]int{}, ArtifactHashes: map[string]string{}}
	if fpErr != nil {
		report.Blockers = append(report.Blockers, "catalog fingerprint failed: "+fpErr.Error())
	}

	manifest, manifestErr := releasegate.LoadManifest(manifestPath)
	currentGitCommit, currentGitErr := resolveCurrentGitCommit()
	if currentGitErr != nil {
		report.Blockers = append(report.Blockers, "current Git commit unavailable: "+currentGitErr.Error())
	}
	if manifestErr != nil {
		report.Blockers = append(report.Blockers, "build manifest unavailable or invalid: "+manifestErr.Error())
	} else {
		report.BuildID = manifest.BuildID
		report.AppVersion = manifest.AppVersion
		report.GitCommit = manifest.GitCommit
		if currentGitErr == nil && !strings.EqualFold(manifest.GitCommit, currentGitCommit) {
			report.Blockers = append(report.Blockers, fmt.Sprintf("build manifest git_commit %s does not match current checkout %s", manifest.GitCommit, currentGitCommit))
		}
		report.CatalogFingerprint = manifest.CatalogFingerprint
		report.WindowsSigningStatus = manifest.WindowsSigningStatus
		for k, a := range manifest.Artifacts {
			report.ArtifactHashes[k] = a.SHA256
		}
		for _, issue := range releaseproof.ValidateManifest(manifest, fingerprint) {
			report.Blockers = append(report.Blockers, "build manifest "+issue.Code+": "+issue.Message)
		}
		artifactIssues := releasegate.ValidateArtifactFiles(manifest, ".")
		writeJSON(filepath.Join(outputDir, "artifact_issues.json"), artifactIssues)
		for _, issue := range artifactIssues {
			report.Blockers = append(report.Blockers, "artifact "+issue.Code+": "+issue.Message)
		}
	}

	key := []byte(os.Getenv(releaseproof.EvidenceKeyEnvironment))
	keyErr := releaseproof.ValidateEvidenceKey(key)
	if keyErr != nil {
		report.Blockers = append(report.Blockers, "evidence authentication unavailable: "+keyErr.Error())
	}

	rs := releasegate.ResultSet{Valid: map[int]releasegate.Result{}}
	if manifestErr == nil && keyErr == nil {
		rs = releasegate.LoadAndValidateResults(resultsDir, manifest, auditEntries, time.Now().UTC(), key)
	} else {
		// Still count files so the report does not silently hide supplied evidence.
		files, _ := filepath.Glob(filepath.Join(resultsDir, "*.json"))
		for _, p := range files {
			base := strings.ToLower(filepath.Base(p))
			if base != "summary.json" && base != "execution_summary.json" && base != "environment_evidence.json" && base != "physical_test_plan.json" && base != "batches.json" {
				rs.Files++
			}
		}
	}
	report.PhysicalResultFiles = rs.Files
	report.ValidPhysicalResults = len(rs.Valid)
	report.InvalidEvidence = rs.Invalid
	report.MismatchedEvidence = rs.Mismatched
	report.DuplicateEvidence = rs.Duplicates
	report.ConflictingEvidence = rs.Conflicts
	writeJSON(filepath.Join(outputDir, "evidence_issues.json"), rs.Issues)
	if rs.Invalid > 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("invalid physical evidence issues: %d", rs.Invalid))
	}
	if rs.Mismatched > 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("mismatched/stale physical evidence issues: %d", rs.Mismatched))
	}
	if rs.Duplicates > 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("duplicate physical evidence files: %d", rs.Duplicates))
	}
	if rs.Conflicts > 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("conflicting physical evidence groups: %d", rs.Conflicts))
	}

	rows := make([]coverageRow, 0, len(auditEntries))
	rootRows := []rootCauseRow{}
	regressionRows := []regressionRow{}
	for _, a := range auditEntries {
		row := coverageRow{Index: a.Index, Name: a.Name, Category: a.Category, CatalogAppID: releaseproof.CatalogAppID(a), Coverage: quality.CoverageUnresolved, RootCause: quality.RootNone, Evidence: "physical Windows install/detect/uninstall/verify result is pending"}
		if a.SystemComponent {
			row.Coverage = quality.CoverageSystemComponent
			row.RootCause = quality.RootSystemComponent
			row.Evidence = "catalog profile explicitly marks a Windows-managed system component"
		} else if a.UninstallStrategy == catalogpkg.StrategyManualOnly {
			row.Coverage = quality.CoverageManualUninstall
			row.RootCause = quality.RootUnsupportedAutomation
			row.Evidence = "catalog profile explicitly requires manual uninstall"
		} else if r, ok := rs.Valid[a.Index]; ok {
			if r.InstallVerified {
				report.VerifiedInstall++
			}
			if r.UninstallVerified {
				report.VerifiedUninstall++
			}
			if a.PhysicalTestRequired && r.FinalStatus != "FULL_PASS" {
				report.FailedPhysical++
			}
			facts := quality.ResultFacts{FinalStatus: r.FinalStatus, FailureStage: r.FailureStage, Failure: r.Failure, SkipReason: r.SkipReason, InstallVerified: r.InstallVerified, UninstallVerified: r.UninstallVerified}
			row.Coverage = quality.CoverageFor(facts)
			row.RootCause = quality.ClassifyRootCause(facts)
			row.PhysicalResult = r.FinalStatus
			row.Evidence = filepath.Join(resultsDir, fmt.Sprintf("%04d.json", a.Index))
			if row.RootCause != quality.RootNone {
				rootRows = append(rootRows, rootCauseRow{Index: a.Index, Name: a.Name, FinalStatus: r.FinalStatus, FailureStage: r.FailureStage, RootCause: row.RootCause, Evidence: row.Evidence})
			}
			if row.Coverage == quality.CoverageVerifiedFull {
				regressionRows = append(regressionRows, regressionRow{Index: a.Index, Name: a.Name, Evidence: row.Evidence, BuildState: "authenticated physical install+detect+uninstall+verify PASS"})
			}
		} else if a.PhysicalTestRequired {
			report.MissingPhysicalEvidence++
		}
		rows = append(rows, row)
		report.RootCauseCounts[string(row.RootCause)]++
		switch row.Coverage {
		case quality.CoverageVerifiedFull:
			report.VerifiedFull++
		case quality.CoverageVerifiedInstallOnly:
			report.VerifiedInstallOnly++
		case quality.CoverageManualUninstall:
			report.ManualUninstall++
		case quality.CoverageSystemComponent:
			report.SystemComponent++
		case quality.CoverageLicenseBlocked:
			report.LicenseBlocked++
		case quality.CoverageUnavailable:
			report.Unavailable++
		default:
			report.Unresolved++
		}
	}

	if !report.CatalogAuditPass {
		report.Blockers = append(report.Blockers, "catalog audit failed")
	}
	if report.UnsafeResolutionEntries != 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("unsafe package resolution entries: %d", report.UnsafeResolutionEntries))
	}
	if report.MissingPhysicalEvidence != 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("required physical evidence missing: %d", report.MissingPhysicalEvidence))
	}
	if report.Unresolved != 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("physical verification unresolved: %d", report.Unresolved))
	}
	if report.VerifiedInstallOnly != 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("install-only physical coverage: %d", report.VerifiedInstallOnly))
	}
	if report.Unavailable != 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("physically unavailable packages require explicit disposition: %d", report.Unavailable))
	}
	if report.LicenseBlocked != 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("license-blocked packages require explicit disposition: %d", report.LicenseBlocked))
	}
	for cause, count := range report.RootCauseCounts {
		if count == 0 || cause == string(quality.RootNone) || cause == string(quality.RootSystemComponent) || cause == string(quality.RootUnsupportedAutomation) {
			continue
		}
		report.Blockers = append(report.Blockers, fmt.Sprintf("unresolved physical root cause %s: %d", cause, count))
	}
	report.ReleaseReady = len(report.Blockers) == 0
	sort.Strings(report.Blockers)

	writeJSON(filepath.Join(outputDir, "release_gate.json"), report)
	writeJSON(filepath.Join(outputDir, "coverage.json"), rows)
	writeCoverageCSV(filepath.Join(outputDir, "coverage.csv"), rows)
	writeJSON(filepath.Join(outputDir, "root_causes.json"), rootRows)
	known := json.RawMessage("[]")
	if b, err := os.ReadFile(filepath.Join("test", "regression", "known_failures.json")); err == nil && json.Valid(b) {
		known = append(json.RawMessage(nil), b...)
	}
	writeJSON(filepath.Join(outputDir, "regression_corpus.json"), regressionCorpus{PhysicalVerified: regressionRows, KnownRegressions: known})
	fmt.Printf("catalog=%d physical_files=%d valid_physical=%d verified_full=%d unresolved=%d invalid=%d mismatched=%d duplicates=%d release_ready=%v\n", report.CatalogTotal, report.PhysicalResultFiles, report.ValidPhysicalResults, report.VerifiedFull, report.Unresolved, report.InvalidEvidence, report.MismatchedEvidence, report.DuplicateEvidence, report.ReleaseReady)
	for _, b := range report.Blockers {
		fmt.Println("BLOCK:", b)
	}
	if !report.ReleaseReady {
		os.Exit(2)
	}
}

func resolveCurrentGitCommit() (string, error) {
	if v := strings.TrimSpace(os.Getenv("WIZ4RDFR0G_GIT_COMMIT")); v != "" {
		if !releaseproof.ValidGitCommit(v) {
			return "", fmt.Errorf("WIZ4RDFR0G_GIT_COMMIT must be a 40-hex Git commit SHA")
		}
		return strings.ToLower(v), nil
	}
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	v := strings.TrimSpace(string(out))
	if !releaseproof.ValidGitCommit(v) {
		return "", fmt.Errorf("git rev-parse HEAD did not return a 40-hex commit SHA")
	}
	return strings.ToLower(v), nil
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0644); err != nil {
		panic(err)
	}
}
func writeCoverageCSV(path string, rows []coverageRow) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"index", "name", "category", "catalog_app_id", "coverage_status", "root_cause", "physical_result", "evidence"})
	for _, r := range rows {
		_ = w.Write([]string{strconv.Itoa(r.Index), r.Name, r.Category, r.CatalogAppID, string(r.Coverage), string(r.RootCause), r.PhysicalResult, r.Evidence})
	}
}
