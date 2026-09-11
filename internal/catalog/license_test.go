package catalog

import "testing"

func TestUnknownLicenseFailsClosed(t *testing.T) {
	m := LicenseFor(AppDef{Name: "Unreviewed Example"}, false)
	if m.Class != LicenseUnknown || m.PolicyOK {
		t.Fatalf("unexpected metadata: %+v", m)
	}
}

func TestOpenSourceRequiresOfficialEvidence(t *testing.T) {
	m := LicenseMetadata{Class: LicenseOpenSource}
	if LicensePolicyAllows(m) {
		t.Fatal("open source without source URL/date must not pass policy")
	}
	m.SourceURL = "https://example.invalid/license"
	m.CheckedAt = "2026-09-11"
	if !LicensePolicyAllows(m) {
		t.Fatal("source-backed open source metadata should pass policy")
	}
}

func TestCommercialNeverPassesFreePolicy(t *testing.T) {
	m := LicenseMetadata{Class: LicenseCommercial, SourceURL: "https://example.invalid/pricing", CheckedAt: "2026-09-11"}
	if LicensePolicyAllows(m) {
		t.Fatal("commercial software must not pass free-app policy")
	}
}
