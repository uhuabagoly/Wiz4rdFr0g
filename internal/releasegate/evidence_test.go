package releasegate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

func fixture(t *testing.T) (releaseproof.BuildManifest, []catalogpkg.AuditEntry, []byte, time.Time) {
	t.Helper()
	entries := catalogpkg.BuildAuditEntries()
	fp, err := releaseproof.CatalogFingerprint(entries)
	if err != nil {
		t.Fatal(err)
	}
	key := []byte(strings.Repeat("k", 32))
	hash := strings.Repeat("a", 64)
	git := strings.Repeat("b", 40)
	m := releaseproof.BuildManifest{SchemaVersion: releaseproof.BuildManifestSchemaVersion, AppVersion: releaseproof.AppVersion, GitCommit: git, CatalogFingerprint: fp, EvidenceSchemaVersion: releaseproof.EvidenceSchemaVersion, Artifacts: map[string]releaseproof.Artifact{releaseproof.PrimaryWindowsArtifactKey: {Path: "dist/Wiz4rdFr0g.exe", SHA256: hash, Size: 1}}, PayloadConsistent: true}
	m.BuildID = releaseproof.BuildID(m.AppVersion, m.GitCommit, m.CatalogFingerprint, hash, m.EvidenceSchemaVersion)
	return m, entries, key, time.Now().UTC()
}
func firstEligibleIndex(entries []catalogpkg.AuditEntry) int {
	for i, e := range entries {
		if e.PhysicalTestRequired && e.LicensePolicyOK {
			return i
		}
	}
	panic("no physical-test-eligible catalog entry")
}

func makeResult(t *testing.T, m releaseproof.BuildManifest, e catalogpkg.AuditEntry, key []byte, now time.Time) Result {
	t.Helper()
	s := releaseproof.EvidenceStatement{SchemaVersion: releaseproof.EvidenceSchemaVersion, BuildID: m.BuildID, AppVersion: m.AppVersion, GitCommit: m.GitCommit, CatalogFingerprint: m.CatalogFingerprint, ArtifactSHA256: m.Artifacts[releaseproof.PrimaryWindowsArtifactKey].SHA256, TestRunID: "run", CatalogIndex: e.Index, CatalogAppID: releaseproof.CatalogAppID(e), CatalogAppName: e.Name, StartedAt: now.Add(-time.Second).Format(time.RFC3339Nano), FinishedAt: now.Format(time.RFC3339Nano), MachineID: "vm", FinalStatus: "FULL_PASS", InstallVerified: true, UninstallVerified: true, DurationSeconds: 1}
	sig, err := releaseproof.EvidenceSignature(s, key)
	if err != nil {
		t.Fatal(err)
	}
	return Result{EvidenceStatement: s, Signature: sig, Name: e.Name}
}
func writeResult(t *testing.T, dir, name string, r Result) {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic unit-test fixture only; these files never enter physical output.
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil { t.Fatal(err) }
	doc["executor"] = "github-actions/windows"
	doc["precheck"] = "CLEAN"
	doc["resolved_id"] = "Unit.Fixture"
	doc["download_ok"], doc["install_ok"], doc["uninstall_ok"], doc["uninstall_attempted"] = true, true, true, true
	doc["download_artifact_present"] = true
	doc["environment"] = map[string]any{"status":"READY", "runner_environment":"github-hosted", "architecture":"X64", "windows_version":"unit fixture", "git_commit":r.GitCommit, "vm_id":r.MachineID}
	doc["independent_before"] = map[string]any{"id":"Unit.Fixture", "state":"absent", "exit_code":-1978335212}
	doc["independent_installed"] = map[string]any{"id":"Unit.Fixture", "state":"present", "exit_code":0}
	doc["independent_removed"] = map[string]any{"id":"Unit.Fixture", "state":"absent", "exit_code":-1978335212}
	b, err = json.Marshal(doc); if err != nil { t.Fatal(err) }
	doc["document_signature"], err = releaseproof.DocumentSignature(b, []byte(strings.Repeat("k",32)))
	if err != nil { t.Fatal(err) }
	b, err = json.Marshal(doc); if err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
		t.Fatal(err)
	}
}
func TestValidResultAccepted(t *testing.T) {
	m, e, key, now := fixture(t)
	dir := t.TempDir()
	writeResult(t, dir, "0000.json", makeResult(t, m, e[firstEligibleIndex(e)], key, now))
	rs := LoadAndValidateResults(dir, m, e, now, key)
	if len(rs.Valid) != 1 || len(rs.Issues) != 0 {
		t.Fatalf("unexpected: %+v", rs)
	}
}
func TestCopiedEvidenceRejected(t *testing.T) {
	m, e, key, now := fixture(t)
	dir := t.TempDir()
	r := makeResult(t, m, e[firstEligibleIndex(e)], key, now)
	writeResult(t, dir, "a.json", r)
	writeResult(t, dir, "b.json", r)
	rs := LoadAndValidateResults(dir, m, e, now, key)
	if len(rs.Valid) != 0 || rs.Duplicates == 0 {
		t.Fatalf("copy accepted: %+v", rs)
	}
}
func TestTamperedEvidenceRejected(t *testing.T) {
	m, e, key, now := fixture(t)
	dir := t.TempDir()
	r := makeResult(t, m, e[firstEligibleIndex(e)], key, now)
	r.TestRunID = "tampered"
	writeResult(t, dir, "x.json", r)
	rs := LoadAndValidateResults(dir, m, e, now, key)
	if len(rs.Valid) != 0 || rs.Invalid == 0 {
		t.Fatalf("tamper accepted: %+v", rs)
	}
}
func TestMissingAndCorruptEvidenceRejected(t *testing.T) {
	m, e, key, now := fixture(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{"), 0644)
	os.WriteFile(filepath.Join(dir, "missing.json"), []byte(`{"catalog_index":0}`), 0644)
	rs := LoadAndValidateResults(dir, m, e, now, key)
	if rs.Invalid < 2 {
		t.Fatalf("expected invalid results: %+v", rs)
	}
}

func TestArtifactPathCannotEscapeProjectRoot(t *testing.T) {
	m, _, _, _ := fixture(t)
	m.Artifacts[releaseproof.WindowsSetupArtifactKey] = releaseproof.Artifact{Path: "../outside.exe", SHA256: strings.Repeat("a", 64), Size: 1}
	m.Artifacts[releaseproof.InstallerPayloadArtifactKey] = releaseproof.Artifact{Path: "payload.exe", SHA256: m.Artifacts[releaseproof.PrimaryWindowsArtifactKey].SHA256, Size: 1}
	m.Artifacts[releaseproof.LinuxAppArtifactKey] = releaseproof.Artifact{Path: "linux", SHA256: strings.Repeat("a", 64), Size: 1}
	m.Artifacts[releaseproof.LinuxPackageArtifactKey] = releaseproof.Artifact{Path: "linux.tar.gz", SHA256: strings.Repeat("a", 64), Size: 1}
	issues := ValidateArtifactFiles(m, t.TempDir())
	found := false
	for _, issue := range issues {
		if issue.Code == "unsafe_artifact_path" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unsafe artifact path was accepted: %+v", issues)
	}
}

func TestPolicyBlockedEntryCannotClaimFullPass(t *testing.T) {
	m, entries, key, now := fixture(t)
	var blocked catalogpkg.AuditEntry
	found := false
	for _, e := range entries {
		if e.PhysicalTestRequired && !e.LicensePolicyOK {
			blocked = e
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing policy-blocked fixture entry")
	}
	dir := t.TempDir()
	r := makeResult(t, m, blocked, key, now)
	writeResult(t, dir, "blocked.json", r)
	rs := LoadAndValidateResults(dir, m, entries, now, key)
	if len(rs.Valid) != 0 || rs.Invalid == 0 {
		t.Fatalf("policy-blocked FULL_PASS was accepted: %+v", rs)
	}
}
