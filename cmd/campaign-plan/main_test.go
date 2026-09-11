package main

import "testing"

func TestPlanPreservesCatalogDenominatorAndDispositions(t *testing.T) {
	p := buildPlan(25)
	if p.CatalogTotal != len(p.Entries) {
		t.Fatalf("catalog total=%d entries=%d", p.CatalogTotal, len(p.Entries))
	}
	if p.PhysicalCandidates != 719 {
		t.Fatalf("physical candidates=%d, want 719; candidate denominator must not shrink because of policy", p.PhysicalCandidates)
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
