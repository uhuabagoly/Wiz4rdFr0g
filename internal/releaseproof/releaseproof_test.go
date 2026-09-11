package releaseproof

import (
	"strings"
	"testing"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

func testManifest(t *testing.T) (BuildManifest, catalogpkg.AuditEntry) {
	t.Helper()
	entries := catalogpkg.BuildAuditEntries()
	fp, err := CatalogFingerprint(entries)
	if err != nil {
		t.Fatal(err)
	}
	art := strings.Repeat("a", 64)
	git := strings.Repeat("b", 40)
	m := BuildManifest{
		SchemaVersion:         BuildManifestSchemaVersion,
		AppVersion:            AppVersion,
		GitCommit:             git,
		CatalogFingerprint:    fp,
		EvidenceSchemaVersion: EvidenceSchemaVersion,
		Artifacts:             map[string]Artifact{PrimaryWindowsArtifactKey: {Path: "dist/Wiz4rdFr0g.exe", SHA256: art, Size: 123}},
		PayloadConsistent:     true,
	}
	m.BuildID = BuildID(m.AppVersion, m.GitCommit, m.CatalogFingerprint, art, m.EvidenceSchemaVersion)
	return m, entries[0]
}

func validStatement(t *testing.T, m BuildManifest, e catalogpkg.AuditEntry, now time.Time, key []byte) (EvidenceStatement, string) {
	t.Helper()
	s := EvidenceStatement{
		SchemaVersion: EvidenceSchemaVersion, BuildID: m.BuildID, AppVersion: m.AppVersion, GitCommit: m.GitCommit,
		CatalogFingerprint: m.CatalogFingerprint, ArtifactSHA256: m.Artifacts[PrimaryWindowsArtifactKey].SHA256,
		TestRunID: "test-run-1", CatalogIndex: e.Index, CatalogAppID: CatalogAppID(e), CatalogAppName: e.Name,
		StartedAt: now.Add(-time.Minute).UTC().Format(time.RFC3339Nano), FinishedAt: now.UTC().Format(time.RFC3339Nano), MachineID: "vm-1",
		FinalStatus: "FULL_PASS", InstallVerified: true, UninstallVerified: true, DurationSeconds: 60,
	}
	sig, err := EvidenceSignature(s, key)
	if err != nil {
		t.Fatal(err)
	}
	return s, sig
}

func TestCatalogFingerprintDeterministic(t *testing.T) {
	entries := catalogpkg.BuildAuditEntries()
	a, err := CatalogFingerprint(entries)
	if err != nil {
		t.Fatal(err)
	}
	b, err := CatalogFingerprint(entries)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("fingerprint changed: %s != %s", a, b)
	}
}

func TestBuildIDChangesForIdentityFields(t *testing.T) {
	base := BuildID("v", strings.Repeat("a", 40), strings.Repeat("b", 64), strings.Repeat("c", 64), 2)
	cases := []string{
		BuildID("v2", strings.Repeat("a", 40), strings.Repeat("b", 64), strings.Repeat("c", 64), 2),
		BuildID("v", strings.Repeat("d", 40), strings.Repeat("b", 64), strings.Repeat("c", 64), 2),
		BuildID("v", strings.Repeat("a", 40), strings.Repeat("e", 64), strings.Repeat("c", 64), 2),
		BuildID("v", strings.Repeat("a", 40), strings.Repeat("b", 64), strings.Repeat("f", 64), 2),
		BuildID("v", strings.Repeat("a", 40), strings.Repeat("b", 64), strings.Repeat("c", 64), 3),
	}
	for i, c := range cases {
		if c == base {
			t.Fatalf("case %d did not change build id", i)
		}
	}
}

func TestEvidenceSignatureRejectsMutation(t *testing.T) {
	m, e := testManifest(t)
	key := []byte(strings.Repeat("k", 32))
	now := time.Now().UTC()
	s, sig := validStatement(t, m, e, now, key)
	if issues := ValidateEvidence(s, sig, m, e, now, key); len(issues) != 0 {
		t.Fatalf("valid evidence rejected: %+v", issues)
	}
	s.TestRunID = "tampered"
	issues := ValidateEvidence(s, sig, m, e, now, key)
	found := false
	for _, i := range issues {
		if i.Code == "signature_invalid" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tampered evidence signature was accepted: %+v", issues)
	}
}

func TestEvidenceRejectsMismatchesAndFutureTime(t *testing.T) {
	m, e := testManifest(t)
	key := []byte(strings.Repeat("k", 32))
	now := time.Now().UTC()
	s, _ := validStatement(t, m, e, now, key)
	s.CatalogAppName = "WRONG"
	s.ArtifactSHA256 = strings.Repeat("0", 64)
	s.StartedAt = now.Add(time.Hour).Format(time.RFC3339Nano)
	s.FinishedAt = now.Add(2 * time.Hour).Format(time.RFC3339Nano)
	sig, err := EvidenceSignature(s, key)
	if err != nil {
		t.Fatal(err)
	}
	issues := ValidateEvidence(s, sig, m, e, now, key)
	want := map[string]bool{"catalog_app_name_mismatch": false, "artifact_hash_mismatch": false, "future_timestamp": false}
	for _, i := range issues {
		if _, ok := want[i.Code]; ok {
			want[i.Code] = true
		}
	}
	for code, ok := range want {
		if !ok {
			t.Fatalf("missing issue %s: %+v", code, issues)
		}
	}
}

func TestManifestValidation(t *testing.T) {
	m, _ := testManifest(t)
	if issues := ValidateManifest(m, m.CatalogFingerprint); len(issues) != 0 {
		t.Fatalf("valid manifest rejected: %+v", issues)
	}
	m.GitCommit = "bad"
	if issues := ValidateManifest(m, m.CatalogFingerprint); len(issues) == 0 {
		t.Fatal("invalid manifest accepted")
	}
}

func TestSystemComponentCannotClaimFullPass(t *testing.T) {
	m, _ := testManifest(t)
	entries := catalogpkg.BuildAuditEntries()
	var system catalogpkg.AuditEntry
	found := false
	for _, e := range entries {
		if e.SystemComponent {
			system = e
			found = true
			break
		}
	}
	if !found {
		t.Fatal("system component fixture not found")
	}
	key := []byte(strings.Repeat("k", 32))
	now := time.Now().UTC()
	s, sig := validStatement(t, m, system, now, key)
	issues := ValidateEvidence(s, sig, m, system, now, key)
	foundIssue := false
	for _, issue := range issues {
		if issue.Code == "invalid_system_component_status" {
			foundIssue = true
		}
	}
	if !foundIssue {
		t.Fatalf("system component FULL_PASS was accepted: %+v", issues)
	}
}
