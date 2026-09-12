package releaseproof

import (
	"encoding/json"
	"fmt"

	"wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

func ValidateTestPlan(b []byte, entries []catalog.AuditEntry) error {
	var p struct {
		Total int `json:"catalog_total"`
		Fingerprint string `json:"catalog_fingerprint"`
		Entries []struct {
			Index int `json:"index"`
			Name string `json:"name"`
			Category string `json:"category"`
			ID string `json:"winget_id"`
			Disposition string `json:"execution_disposition"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(b,&p); err != nil { return err }
	fp, err := CatalogFingerprint(entries); if err != nil { return err }
	if p.Total != len(entries) || len(p.Entries) != len(entries) || p.Fingerprint != fp { return fmt.Errorf("plan coverage/fingerprint mismatch") }
	seen := map[int]bool{}
	for _, e := range p.Entries {
		if e.Index < 0 || e.Index >= len(entries) || seen[e.Index] { return fmt.Errorf("duplicate/out-of-range plan index") }
		seen[e.Index] = true
		want := entries[e.Index]
		disposition := "NOT_PHYSICAL"
		switch {
		case want.SystemComponent: disposition = "SYSTEM_COMPONENT"
		case want.UninstallStrategy == catalog.StrategyManualOnly: disposition = "MANUAL_ONLY"
		case !want.LicensePolicyOK: disposition = "POLICY_BLOCKED"
		case want.PhysicalTestRequired: disposition = "PHYSICAL_REQUIRED"
		}
		if e.Name != want.Name || e.ID != want.WingetID || e.Category != want.Category || e.Disposition != disposition { return fmt.Errorf("plan identity/disposition mismatch at %d", e.Index) }
	}
	return nil
}
