package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

type planEntry struct {
	Index                int                     `json:"index"`
	Name                 string                  `json:"name"`
	Category             string                  `json:"category"`
	WingetID             string                  `json:"winget_id,omitempty"`
	LicenseClass         catalogpkg.LicenseClass `json:"license_class"`
	LicensePolicyOK      bool                    `json:"license_policy_ok"`
	PhysicalCandidate    bool                    `json:"physical_candidate"`
	ExecutionDisposition string                  `json:"execution_disposition"`
}

type batch struct {
	Batch   int   `json:"batch"`
	Indexes []int `json:"indexes"`
}

type plan struct {
	GeneratedAt        string      `json:"generated_at"`
	CatalogTotal       int         `json:"catalog_total"`
	PhysicalCandidates int         `json:"physical_candidates"`
	PhysicalEligible   int         `json:"physical_eligible"`
	PolicyBlocked      int         `json:"policy_blocked"`
	SystemComponents   int         `json:"system_components"`
	ManualOnly         int         `json:"manual_only"`
	Entries            []planEntry `json:"entries"`
	Batches            []batch     `json:"batches"`
}

func buildPlan(batchSize int) plan {
	if batchSize < 1 {
		batchSize = 25
	}
	entries := catalogpkg.BuildAuditEntries()
	p := plan{GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), CatalogTotal: len(entries)}
	var eligible []int
	for _, e := range entries {
		disposition := "NOT_PHYSICAL"
		if e.SystemComponent {
			disposition = "SYSTEM_COMPONENT"
			p.SystemComponents++
		} else if e.UninstallStrategy == catalogpkg.StrategyManualOnly {
			disposition = "MANUAL_ONLY"
			p.ManualOnly++
		} else if !e.LicensePolicyOK {
			disposition = "POLICY_BLOCKED"
			p.PolicyBlocked++
		} else if e.PhysicalTestRequired {
			disposition = "PHYSICAL_REQUIRED"
			p.PhysicalEligible++
			eligible = append(eligible, e.Index)
		}
		if e.PhysicalTestRequired {
			p.PhysicalCandidates++
		}
		p.Entries = append(p.Entries, planEntry{Index: e.Index, Name: e.Name, Category: e.Category, WingetID: e.WingetID, LicenseClass: e.LicenseClass, LicensePolicyOK: e.LicensePolicyOK, PhysicalCandidate: e.PhysicalTestRequired, ExecutionDisposition: disposition})
	}
	for i := 0; i < len(eligible); i += batchSize {
		end := i + batchSize
		if end > len(eligible) {
			end = len(eligible)
		}
		p.Batches = append(p.Batches, batch{Batch: len(p.Batches), Indexes: append([]int(nil), eligible[i:end]...)})
	}
	return p
}

func main() {
	outDir := "test/windows-vm"
	if len(os.Args) > 1 && os.Args[1] != "" {
		outDir = os.Args[1]
	}
	p := buildPlan(25)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}
	writeJSON(filepath.Join(outDir, "physical_test_plan.json"), p)
	writeJSON(filepath.Join(outDir, "batches.json"), p.Batches)
	fmt.Printf("catalog=%d physical_candidates=%d physical_eligible=%d policy_blocked=%d system=%d manual=%d batches=%d\n", p.CatalogTotal, p.PhysicalCandidates, p.PhysicalEligible, p.PolicyBlocked, p.SystemComponents, p.ManualOnly, len(p.Batches))
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		panic(err)
	}
}
