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
	if err := os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
		t.Fatal(err)
	}
}
func TestValidResultAccepted(t *testing.T) {
	m, e, key, now := fixture(t)
	dir := t.TempDir()
	writeResult(t, dir, "0000.json", makeResult(t, m, e[0], key, now))
	rs := LoadAndValidateResults(dir, m, e, now, key)
	if len(rs.Valid) != 1 || len(rs.Issues) != 0 {
		t.Fatalf("unexpected: %+v", rs)
	}
}
func TestCopiedEvidenceRejected(t *testing.T) {
	m, e, key, now := fixture(t)
	dir := t.TempDir()
	r := makeResult(t, m, e[0], key, now)
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
	r := makeResult(t, m, e[0], key, now)
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
