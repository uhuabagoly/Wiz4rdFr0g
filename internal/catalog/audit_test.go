package catalog

import "testing"

func TestCatalogAuditCoverage(t *testing.T) {
	entries := BuildAuditEntries()
	if len(entries) != 723 {
		t.Fatalf("catalog entries = %d, want 723", len(entries))
	}
	s := Summarize(entries)
	if !s.SafetyPass {
		t.Fatalf("catalog safety audit failed: %v", s.ValidationErrors)
	}
	if s.LicensePolicyPass {
		t.Fatal("license policy unexpectedly passed before all 723 entries have official-source classification")
	}
	if s.LicenseCommercial != 4 {
		t.Fatalf("commercial license entries=%d, want 4", s.LicenseCommercial)
	}
	if s.LicensePolicyEligible < 3 {
		t.Fatalf("policy eligible entries=%d, want at least the source-verified open-source seed set", s.LicensePolicyEligible)
	}
	if s.DuplicateNames != 0 {
		t.Fatalf("duplicate names = %d", s.DuplicateNames)
	}
	if s.HardcodedIDEntries != 74 {
		t.Fatalf("hardcoded ID entries = %d, want 74", s.HardcodedIDEntries)
	}
	if s.VerifiedHardcodedIDEntries != 74 {
		t.Fatalf("verified hardcoded ID entries = %d, want 74", s.VerifiedHardcodedIDEntries)
	}
	if s.UnverifiedHardcodedIDEntries != 0 {
		t.Fatalf("unverified hardcoded ID entries = %d, want 0", s.UnverifiedHardcodedIDEntries)
	}
	if s.ExactNameVerifiedEntries != 228 {
		t.Fatalf("exact-name verified entries = %d, want 228", s.ExactNameVerifiedEntries)
	}
	if s.RuntimeExactRequiredEntries != 421 {
		t.Fatalf("runtime-exact entries = %d, want 421", s.RuntimeExactRequiredEntries)
	}
	if s.UnsafeResolutionEntries != 0 {
		t.Fatalf("unsafe resolution entries = %d", s.UnsafeResolutionEntries)
	}
}

func TestCatalogAliasesAreExplicit(t *testing.T) {
	profiles := map[string]Profile{}
	for _, app := range Entries {
		profiles[app.Name] = ProfileFor(app)
	}
	if profiles["Python"].CatalogAliasOf != "Python 3" {
		t.Fatalf("Python alias = %q", profiles["Python"].CatalogAliasOf)
	}
	if profiles["KeePass"].CatalogAliasOf != "KeePass 2" {
		t.Fatalf("KeePass alias = %q", profiles["KeePass"].CatalogAliasOf)
	}
}

func TestSystemComponentsAreNotAutomaticUninstall(t *testing.T) {
	for _, name := range []string{"Microsoft Edge", "OpenSSH for Windows", "Windows Defender"} {
		var found bool
		for _, app := range Entries {
			if app.Name != name {
				continue
			}
			found = true
			p := ProfileFor(app)
			if !p.SystemComponent || p.UninstallStrategy != StrategySystemComponentUnsupported || p.UninstallSupport != SupportUnsupported {
				t.Fatalf("unsafe system component profile for %s: %+v", name, p)
			}
		}
		if !found {
			t.Fatalf("missing system component %s", name)
		}
	}
}

func TestDBeaverCurrentAndHistoricalIDs(t *testing.T) {
	for _, app := range Entries {
		if app.Name != "DBeaver" {
			continue
		}
		p := ProfileFor(app)
		if p.WingetID != "DBeaver.DBeaver.Community" {
			t.Fatalf("current DBeaver ID = %q", p.WingetID)
		}
		if len(p.AlternativeIDs) != 1 || p.AlternativeIDs[0] != "dbeaver.dbeaver" {
			t.Fatalf("historical DBeaver IDs = %v", p.AlternativeIDs)
		}
		return
	}
	t.Fatal("DBeaver missing from catalog")
}

func TestPhysicalTestRequiredExcludesSystemAndManualOnly(t *testing.T) {
	entries := BuildAuditEntries()
	summary := Summarize(entries)
	if summary.PhysicalTestRequired != 719 {
		t.Fatalf("physical test required=%d, want 719", summary.PhysicalTestRequired)
	}
	for _, e := range entries {
		if (e.SystemComponent || e.UninstallStrategy == StrategyManualOnly) && e.PhysicalTestRequired {
			t.Fatalf("non-automatable entry %q is incorrectly marked physical-test-required", e.Name)
		}
	}
}
