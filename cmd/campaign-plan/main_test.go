package main

import (
	"testing"
	"encoding/json"
	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

func TestPlanGateRejectsMissingDuplicateAndStaleEntries(t *testing.T) {
	p := buildPlan(25)
	b, _ := json.Marshal(p)
	entries := catalogpkg.BuildAuditEntries()
	if err := releaseproof.ValidateTestPlan(b,entries); err != nil { t.Fatal(err) }
	for _, kind := range []string{"missing","duplicate","fingerprint","status"} {
		p := buildPlan(25)
		switch kind {
		case "missing": p.Entries = p.Entries[1:]
		case "duplicate": p.Entries[1].Index = p.Entries[0].Index
		case "fingerprint": p.CatalogFingerprint = "stale"
		case "status": p.Entries[0].ExecutionDisposition = "MANUAL_ONLY"
		}
		b,_ := json.Marshal(p)
		if releaseproof.ValidateTestPlan(b,entries) == nil { t.Fatalf("accepted %s plan",kind) }
	}
}

func TestPlanPreservesCatalogDenominatorAndDispositions(t *testing.T) {
	p := buildPlan(25)
	if p.CatalogTotal != len(p.Entries) {
		t.Fatalf("catalog total=%d entries=%d", p.CatalogTotal, len(p.Entries))
	}
	want := 0
	for _, e := range catalogpkg.BuildAuditEntries() { if e.PhysicalTestRequired { want++ } }
	if p.PhysicalCandidates != want {
		t.Fatalf("physical candidates=%d, want %d; candidate denominator must not shrink because of policy", p.PhysicalCandidates, want)
	}
	if p.SystemComponents != 3 {
		t.Fatalf("system components=%d, want 3", p.SystemComponents)
	}
	if p.ManualOnly != 1 {
		t.Fatalf("manual-only=%d, want 1", p.ManualOnly)
	}
	if p.PhysicalEligible+p.PolicyBlocked+p.SystemComponents+p.ManualOnly != p.CatalogTotal {
		t.Fatalf("dispositions do not cover catalog: eligible=%d blocked=%d system=%d manual=%d total=%d", p.PhysicalEligible, p.PolicyBlocked, p.SystemComponents, p.ManualOnly, p.CatalogTotal)
	}
	if p.PhysicalEligible == 0 {
		t.Fatal("expected at least the source-verified seed entries to be physically eligible")
	}
}

func TestPlanBatchesContainOnlyEligibleEntries(t *testing.T) {
	p := buildPlan(2)
	eligible := map[int]bool{}
	for _, e := range p.Entries {
		if e.ExecutionDisposition == "PHYSICAL_REQUIRED" {
			eligible[e.Index] = true
		}
	}
	seen := map[int]bool{}
	for _, b := range p.Batches {
		for _, idx := range b.Indexes {
			if !eligible[idx] {
				t.Fatalf("batch contains non-eligible catalog index %d", idx)
			}
			if seen[idx] {
				t.Fatalf("eligible catalog index %d appears in multiple batches", idx)
			}
			seen[idx] = true
		}
	}
	if len(seen) != p.PhysicalEligible {
		t.Fatalf("batched=%d physical eligible=%d", len(seen), p.PhysicalEligible)
	}
}
