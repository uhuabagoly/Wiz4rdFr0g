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

	"wiz4rdfr0g.local/fullcatalog/internal/quality"
)

type result struct {
	CatalogIndex       int               `json:"catalog_index"`
	Name               string            `json:"name"`
	ResolvedID         string            `json:"resolved_id"`
	DownloadRetryCount int               `json:"download_retry_count"`
	InstallRetryCount  int               `json:"install_retry_count"`
	UninstallDiagnosis string            `json:"uninstall_diagnosis"`
	UninstallAttempts  []json.RawMessage `json:"uninstall_attempts"`
	InstallOK          bool              `json:"install_ok"`
	InstallVerified    bool              `json:"install_verified"`
	UninstallOK        bool              `json:"uninstall_ok"`
	UninstallVerified  bool              `json:"uninstall_verified"`
	FinalStatus        string            `json:"final_status"`
	RootCause          string            `json:"root_cause"`
	CoverageStatus     string            `json:"coverage_status"`
	SkipReason         string            `json:"skip_reason"`
	FailureStage       string            `json:"failure_stage"`
	Failure            string            `json:"failure"`
}

type summary struct {
	TotalResults        int            `json:"total_results"`
	Tested              int            `json:"tested"`
	FullPass            int            `json:"full_pass"`
	InstallFail         int            `json:"install_fail"`
	DetectionFail       int            `json:"detection_fail"`
	UninstallFail       int            `json:"uninstall_fail"`
	UninstallRepairFail int            `json:"uninstall_repair_fail"`
	VerifyFail          int            `json:"verify_fail"`
	Ambiguous           int            `json:"ambiguous"`
	Unsupported         int            `json:"unsupported"`
	NotExecuted         int            `json:"not_executed"`
	RetryEvents         int            `json:"retry_events"`
	ManualOnly          int            `json:"manual_only"`
	SystemComponent     int            `json:"system_component"`
	LicenseRequired     int            `json:"license_required"`
	Unavailable         int            `json:"unavailable"`
	OtherSkipped        int            `json:"other_skipped"`
	Other               int            `json:"other"`
	StatusCounts        map[string]int `json:"status_counts"`
	RootCauseCounts     map[string]int `json:"root_cause_counts"`
	CoverageCounts      map[string]int `json:"coverage_counts"`
}

func main() {
	dir := "test/windows-vm/results"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		panic(err)
	}
	var rows []result
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f))
		if base == "summary.json" || base == "execution_summary.json" || base == "environment_evidence.json" || base == "physical_test_plan.json" || base == "batches.json" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			panic(err)
		}
		var r result
		if err := json.Unmarshal(b, &r); err != nil {
			panic(fmt.Errorf("%s: %w", f, err))
		}
		if r.FinalStatus == "" {
			continue
		}
		facts := quality.ResultFacts{FinalStatus: r.FinalStatus, FailureStage: r.FailureStage, Failure: r.Failure, SkipReason: r.SkipReason, InstallVerified: r.InstallVerified, UninstallVerified: r.UninstallVerified}
		if r.RootCause == "" {
			r.RootCause = string(quality.ClassifyRootCause(facts))
		}
		if r.CoverageStatus == "" {
			r.CoverageStatus = string(quality.CoverageFor(facts))
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CatalogIndex < rows[j].CatalogIndex })
	s := summary{TotalResults: len(rows), StatusCounts: map[string]int{}, RootCauseCounts: map[string]int{}, CoverageCounts: map[string]int{}}
	for _, r := range rows {
		s.StatusCounts[r.FinalStatus]++
		s.RetryEvents += r.DownloadRetryCount + r.InstallRetryCount
		if len(r.UninstallAttempts) > 1 {
			s.RetryEvents += len(r.UninstallAttempts) - 1
		}
		s.RootCauseCounts[r.RootCause]++
		s.CoverageCounts[r.CoverageStatus]++
		switch r.FinalStatus {
		case "FULL_PASS":
			s.Tested++
			s.FullPass++
		case "INSTALL_FAIL":
			s.Tested++
			s.InstallFail++
		case "DETECTION_FAIL":
			s.Tested++
			s.DetectionFail++
		case "UNINSTALL_FAIL":
			s.Tested++
			s.UninstallFail++
		case "UNINSTALL_REPAIR_FAILED":
			s.Tested++
			s.UninstallRepairFail++
		case "VERIFY_FAIL":
			s.Tested++
			s.VerifyFail++
		case "MANUAL_ONLY":
			s.ManualOnly++
		case "SYSTEM_COMPONENT":
			s.SystemComponent++
		case "LICENSE_REQUIRED":
			s.LicenseRequired++
		case "UNAVAILABLE":
			s.Unavailable++
		case "AMBIGUOUS":
			s.Tested++
			s.Ambiguous++
		case "UNSUPPORTED":
			s.Unsupported++
		case "NOT_EXECUTED":
			s.NotExecuted++
		default:
			if strings.HasPrefix(r.FinalStatus, "SKIPPED_") {
				s.OtherSkipped++
			} else {
				s.Other++
			}
		}
	}
	sb, _ := json.MarshalIndent(s, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), append(sb, '\n'), 0644); err != nil {
		panic(err)
	}
	f, err := os.Create(filepath.Join(dir, "summary.csv"))
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"catalog_index", "name", "resolved_id", "download_retries", "install_retries", "uninstall_attempts", "uninstall_diagnosis", "install_ok", "install_verified", "uninstall_ok", "uninstall_verified", "final_status", "coverage_status", "root_cause", "failure_stage", "failure", "skip_reason"})
	for _, r := range rows {
		_ = w.Write([]string{strconv.Itoa(r.CatalogIndex), r.Name, r.ResolvedID, strconv.Itoa(r.DownloadRetryCount), strconv.Itoa(r.InstallRetryCount), strconv.Itoa(len(r.UninstallAttempts)), r.UninstallDiagnosis, strconv.FormatBool(r.InstallOK), strconv.FormatBool(r.InstallVerified), strconv.FormatBool(r.UninstallOK), strconv.FormatBool(r.UninstallVerified), r.FinalStatus, r.CoverageStatus, r.RootCause, r.FailureStage, r.Failure, r.SkipReason})
	}
	w.Flush()
	_ = f.Close()
	fmt.Println(string(sb))
}
