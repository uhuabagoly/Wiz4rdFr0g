// actions-gate validates the complete eligible Windows campaign; it never declares a full release ready.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/releasegate"
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

func main() {
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "usage: actions-gate manifest plan pilot-indexes results-dir github-RUN-ATTEMPT")
		os.Exit(2)
	}
	if err := check(os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("CAMPAIGN_PASS: all selected physical results authenticated; full release remains subject to the catalog release gate")
}

func read(path string, value any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, value)
}

func check(manifestPath, planPath, pilotPath, dir, run string) error {
	m, err := releasegate.LoadManifest(manifestPath)
	if err != nil {
		return err
	}
	entries := catalog.BuildAuditEntries()
	fp, err := releaseproof.CatalogFingerprint(entries)
	if err != nil {
		return err
	}
	if issues := releaseproof.ValidateManifest(m, fp); len(issues) != 0 {
		return fmt.Errorf("manifest invalid: %+v", issues)
	}
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	if err := releaseproof.ValidateTestPlan(planBytes, entries); err != nil {
		return err
	}
	var p struct {
		Total       int    `json:"catalog_total"`
		Fingerprint string `json:"catalog_fingerprint"`
		Entries     []struct {
			Index    int    `json:"index"`
			Name     string `json:"name"`
			Category string `json:"category"`
			ID       string `json:"winget_id"`
		} `json:"entries"`
	}
	if err := read(planPath, &p); err != nil {
		return err
	}
	if p.Total != len(entries) || len(p.Entries) != len(entries) || p.Fingerprint != fp {
		return fmt.Errorf("plan coverage/fingerprint mismatch")
	}
	seen := map[int]bool{}
	for _, e := range p.Entries {
		if e.Index < 0 || e.Index >= len(entries) || seen[e.Index] {
			return fmt.Errorf("duplicate/out-of-range plan index")
		}
		seen[e.Index] = true
		want := entries[e.Index]
		if e.Name != want.Name || e.ID != want.WingetID || e.Category != want.Category {
			return fmt.Errorf("plan identity mismatch at %d", e.Index)
		}
	}
	var selected []int
	if err := read(pilotPath, &selected); err != nil {
		return err
	}
	if err := validateSelection(entries, selected); err != nil {
		return err
	}
	key := []byte(os.Getenv(releaseproof.EvidenceKeyEnvironment))
	results := releasegate.LoadAndValidateResults(dir, m, entries, time.Now().UTC(), key)
	if len(results.Issues) != 0 {
		return fmt.Errorf("evidence issues: %+v", results.Issues)
	}
	if len(results.Valid) != len(selected) {
		return fmt.Errorf("missing or unexpected result count")
	}
	seen = map[int]bool{}
	for _, idx := range selected {
		if idx < 0 || idx >= len(entries) || seen[idx] {
			return fmt.Errorf("invalid pilot index")
		}
		seen[idx] = true
		r, ok := results.Valid[idx]
		if !ok || r.FinalStatus != "FULL_PASS" || r.TestRunID != run || r.MachineID != fmt.Sprintf("%s-%d", run, idx) {
			return fmt.Errorf("index %d missing FULL_PASS or current run/VM identity", idx)
		}
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%04d.json", idx)))
		if err != nil {
			return err
		}
		if err := releaseproof.ValidatePhysicalDocument(b, key); err != nil {
			return fmt.Errorf("index %d: %w", idx, err)
		}
	}
	return nil
}

func validateSelection(entries []catalog.AuditEntry, selected []int) error {
	required := map[int]bool{}
	for _, e := range entries {
		if !e.SystemComponent && e.UninstallStrategy != catalog.StrategyManualOnly && e.LicensePolicyOK && e.PhysicalTestRequired {
			required[e.Index] = true
		}
	}
	if len(selected) == 0 || len(selected) != len(required) {
		return fmt.Errorf("selection must cover every eligible catalog entry")
	}
	selectedSeen := map[int]bool{}
	for _, idx := range selected {
		if !required[idx] || selectedSeen[idx] {
			return fmt.Errorf("unexpected or duplicate selected index %d", idx)
		}
		selectedSeen[idx] = true
	}
	return nil
}
