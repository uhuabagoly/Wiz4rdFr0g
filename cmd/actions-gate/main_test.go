package main

import (
	"testing"
	"wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

func TestCampaignCannotOmitEligibleEntries(t *testing.T) {
	entries := catalog.BuildAuditEntries()
	var all []int
	for _, e := range entries {
		if !e.SystemComponent && e.UninstallStrategy != catalog.StrategyManualOnly && e.LicensePolicyOK && e.PhysicalTestRequired {
			all = append(all, e.Index)
		}
	}
	if len(all) < 2 {
		t.Fatal("fixture needs multiple eligible entries")
	}
	if err := validateSelection(entries, all); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]int{nil, all[:len(all)-1], append(append([]int{}, all...), all[0]), {-1}} {
		if err := validateSelection(entries, bad); err == nil {
			t.Fatalf("incomplete/invalid selection accepted: %v", bad)
		}
	}
	duplicate := append([]int{}, all...)
	duplicate[1] = duplicate[0]
	if err := validateSelection(entries, duplicate); err == nil {
		t.Fatal("same-sized duplicated selection accepted")
	}
}
