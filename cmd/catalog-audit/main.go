package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

func main() {
	outDir := flag.String("out", "audit", "")
	flag.Parse()
	entries := catalogpkg.BuildAuditEntries()
	summary := catalogpkg.Summarize(entries)
	report := catalogpkg.AuditReport{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Scope: "static catalog, package-resolution safety and source-backed license-policy audit; physical Windows install/uninstall is reported separately", Summary: summary, Entries: entries}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(err)
	}
	writeJSON(filepath.Join(*outDir, "catalog_audit.json"), report)
	writeJSON(filepath.Join(*outDir, "summary.json"), summary)
	writeCSV(filepath.Join(*outDir, "catalog_audit.csv"), entries)
	fmt.Printf("catalog=%d canonical=%d aliases=%d hardcoded_ids=%d verified_hardcoded=%d exact_name_verified=%d runtime_exact=%d unresolved=%d unsafe=%d license_ok=%d license_blocked=%d license_unknown=%d safety_pass=%t policy_pass=%t pass=%t\n", summary.CatalogEntries, summary.CanonicalPrograms, summary.CatalogAliases, summary.HardcodedIDEntries, summary.VerifiedHardcodedIDEntries, summary.ExactNameVerifiedEntries, summary.RuntimeExactRequiredEntries, summary.Unresolved, summary.UnsafeResolutionEntries, summary.LicensePolicyEligible, summary.LicensePolicyBlocked, summary.LicenseUnknown, summary.SafetyPass, summary.LicensePolicyPass, summary.Pass)
	if !summary.Pass {
		os.Exit(1)
	}
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		panic(err)
	}
}

func writeCSV(path string, entries []catalogpkg.AuditEntry) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"index", "name", "category", "canonical_name", "alias_of", "platform", "winget_id", "winget_id_verified", "resolution_status", "package_resolution_safe", "detection_strategy", "expected_scope", "installer_type", "uninstall_strategy", "uninstall_support", "system_component", "physical_test_required", "license_class", "license_policy_ok", "license_source_url", "license_checked_at", "license_note", "unresolved_reason"})
	for _, e := range entries {
		_ = w.Write([]string{strconv.Itoa(e.Index), e.Name, e.Category, e.CanonicalName, e.CatalogAliasOf, string(e.Platform), e.WingetID, strconv.FormatBool(e.WingetIDVerified), string(e.ResolutionStatus), strconv.FormatBool(e.PackageResolutionSafe), e.DetectionStrategy, string(e.ExpectedScope), string(e.InstallerType), string(e.UninstallStrategy), string(e.UninstallSupport), strconv.FormatBool(e.SystemComponent), strconv.FormatBool(e.PhysicalTestRequired), string(e.LicenseClass), strconv.FormatBool(e.LicensePolicyOK), e.LicenseSourceURL, e.LicenseCheckedAt, e.LicenseNote, e.UnresolvedReason})
	}
	if err := w.Error(); err != nil {
		panic(err)
	}
}
