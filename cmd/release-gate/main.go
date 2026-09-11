package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/quality"
)

type vmResult struct {
	CatalogIndex      int    `json:"catalog_index"`
	Name              string `json:"name"`
	FinalStatus       string `json:"final_status"`
	FailureStage      string `json:"failure_stage"`
	Failure           string `json:"failure"`
	SkipReason        string `json:"skip_reason"`
	InstallVerified   bool   `json:"install_verified"`
	UninstallVerified bool   `json:"uninstall_verified"`
}

type coverageRow struct {
	Index          int                    `json:"index"`
	Name           string                 `json:"name"`
	Category       string                 `json:"category"`
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
	GeneratedAt             string         `json:"generated_at"`
	CatalogTotal            int            `json:"catalog_total"`
	CatalogAuditPass        bool           `json:"catalog_audit_pass"`
	UnsafeResolutionEntries int            `json:"unsafe_resolution_entries"`
	PhysicalResults         int            `json:"physical_results"`
	VerifiedFull            int            `json:"verified_full"`
	VerifiedInstallOnly     int            `json:"verified_install_only"`
	ManualUninstall         int            `json:"manual_uninstall"`
	SystemComponent         int            `json:"system_component"`
	LicenseBlocked          int            `json:"license_blocked"`
	Unavailable             int            `json:"unavailable"`
	Unresolved              int            `json:"unresolved"`
	RootCauseCounts         map[string]int `json:"root_cause_counts"`
	ReleaseReady            bool           `json:"release_ready"`
	Blockers                []string       `json:"blockers"`
}

func main() {
	resultsDir := "test/windows-vm/results"
	outputDir := "release"
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		resultsDir = os.Args[1]
	}
	if len(os.Args) > 2 && strings.TrimSpace(os.Args[2]) != "" {
		outputDir = os.Args[2]
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	auditEntries := catalogpkg.BuildAuditEntries()
	auditSummary := catalogpkg.Summarize(auditEntries)
	results := loadResults(resultsDir)
	byIndex := map[int]vmResult{}
	for _, r := range results {
		byIndex[r.CatalogIndex] = r
	}

	rows := make([]coverageRow, 0, len(auditEntries))
	rootRows := make([]rootCauseRow, 0)
	regressionRows := make([]regressionRow, 0)
	report := gateReport{
		GeneratedAt:             time.Now().UTC().Format(time.RFC3339),
		CatalogTotal:            len(auditEntries),
		CatalogAuditPass:        auditSummary.Pass,
		UnsafeResolutionEntries: auditSummary.UnsafeResolutionEntries,
		PhysicalResults:         len(results),
		RootCauseCounts:         map[string]int{},
	}
	for _, a := range auditEntries {
		row := coverageRow{Index: a.Index, Name: a.Name, Category: a.Category, Coverage: quality.CoverageUnresolved, RootCause: quality.RootNone, Evidence: "physical Windows install/detect/uninstall/verify result is pending"}
		if r, ok := byIndex[a.Index]; ok {
			facts := quality.ResultFacts{FinalStatus: r.FinalStatus, FailureStage: r.FailureStage, Failure: r.Failure, SkipReason: r.SkipReason, InstallVerified: r.InstallVerified, UninstallVerified: r.UninstallVerified}
			row.Coverage = quality.CoverageFor(facts)
			row.RootCause = quality.ClassifyRootCause(facts)
			row.PhysicalResult = r.FinalStatus
			row.Evidence = filepath.Join(resultsDir, fmt.Sprintf("%04d.json", a.Index))
			if row.RootCause != quality.RootNone {
				rootRows = append(rootRows, rootCauseRow{Index: a.Index, Name: a.Name, FinalStatus: r.FinalStatus, FailureStage: r.FailureStage, RootCause: row.RootCause, Evidence: row.Evidence})
			}
			if row.Coverage == quality.CoverageVerifiedFull {
				regressionRows = append(regressionRows, regressionRow{Index: a.Index, Name: a.Name, Evidence: row.Evidence, BuildState: "physical install+detect+uninstall+verify PASS"})
			}
		} else if a.SystemComponent {
			row.Coverage = quality.CoverageSystemComponent
			row.RootCause = quality.RootSystemComponent
			row.Evidence = "catalog profile explicitly marks a Windows-managed system component"
		} else if a.UninstallStrategy == catalogpkg.StrategyManualOnly {
			row.Coverage = quality.CoverageManualUninstall
			row.RootCause = quality.RootUnsupportedAutomation
			row.Evidence = "catalog profile explicitly requires manual uninstall"
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
	fmt.Printf("catalog=%d physical=%d verified_full=%d unresolved=%d release_ready=%v\n", report.CatalogTotal, report.PhysicalResults, report.VerifiedFull, report.Unresolved, report.ReleaseReady)
	for _, b := range report.Blockers {
		fmt.Println("BLOCK:", b)
	}
	if !report.ReleaseReady {
		os.Exit(2)
	}
}

func loadResults(dir string) []vmResult {
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var out []vmResult
	for _, path := range files {
		base := strings.ToLower(filepath.Base(path))
		if base == "summary.json" || base == "execution_summary.json" || base == "environment_evidence.json" || base == "physical_test_plan.json" || base == "batches.json" {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var r vmResult
		if json.Unmarshal(b, &r) == nil && r.FinalStatus != "" && r.CatalogIndex >= 0 {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CatalogIndex < out[j].CatalogIndex })
	return out
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
	_ = w.Write([]string{"index", "name", "category", "coverage_status", "root_cause", "physical_result", "evidence"})
	for _, r := range rows {
		_ = w.Write([]string{strconv.Itoa(r.Index), r.Name, r.Category, string(r.Coverage), string(r.RootCause), r.PhysicalResult, r.Evidence})
	}
}
