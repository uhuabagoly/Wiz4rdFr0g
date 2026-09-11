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

func resign(t *testing.T, r *Result, key []byte) {
	t.Helper()
	sig, err := releaseproof.EvidenceSignature(r.EvidenceStatement, key)
	if err != nil {
		t.Fatal(err)
	}
	r.Signature = sig
}

func assertRejected(t *testing.T, dir string, m releaseproof.BuildManifest, entries []catalogpkg.AuditEntry, now time.Time, key []byte) ResultSet {
	t.Helper()
	rs := LoadAndValidateResults(dir, m, entries, now, key)
	if len(rs.Valid) != 0 {
		t.Fatalf("adversarial evidence accepted: %+v", rs)
	}
	if len(rs.Issues) == 0 {
		t.Fatalf("adversarial evidence rejected without audit issue: %+v", rs)
	}
	return rs
}

func TestAdversarialEvidenceMatrix(t *testing.T) {
	baseManifest, entries, key, now := fixture(t)
	cases := []struct {
		name   string
		mutate func(*testing.T, *Result, []byte)
		raw    func(*testing.T, string, Result)
	}{
		{name: "fake_full_pass_without_signature", mutate: func(t *testing.T, r *Result, key []byte) { r.Signature = "" }},
		{name: "wrong_application_name", mutate: func(t *testing.T, r *Result, key []byte) { r.CatalogAppName = "WRONG-NAME"; resign(t, r, key) }},
		{name: "correct_index_wrong_app_id", mutate: func(t *testing.T, r *Result, key []byte) {
			r.CatalogAppID = "catalog-entry-sha256:" + strings.Repeat("0", 64)
			resign(t, r, key)
		}},
		{name: "correct_app_identity_wrong_index", mutate: func(t *testing.T, r *Result, key []byte) { r.CatalogIndex = 1; resign(t, r, key) }},
		{name: "old_git_commit", mutate: func(t *testing.T, r *Result, key []byte) { r.GitCommit = strings.Repeat("c", 40); resign(t, r, key) }},
		{name: "other_catalog_fingerprint", mutate: func(t *testing.T, r *Result, key []byte) {
			r.CatalogFingerprint = strings.Repeat("d", 64)
			resign(t, r, key)
		}},
		{name: "other_artifact_sha256", mutate: func(t *testing.T, r *Result, key []byte) {
			r.ArtifactSHA256 = strings.Repeat("e", 64)
			resign(t, r, key)
		}},
		{name: "old_schema", mutate: func(t *testing.T, r *Result, key []byte) { r.SchemaVersion = 1; resign(t, r, key) }},
		{name: "unknown_schema", mutate: func(t *testing.T, r *Result, key []byte) { r.SchemaVersion = 999; resign(t, r, key) }},
		{name: "corrupt_json", raw: func(t *testing.T, dir string, r Result) {
			if err := os.WriteFile(filepath.Join(dir, "0000.json"), []byte("{not-json"), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "missing_required_field", raw: func(t *testing.T, dir string, r Result) {
			b, _ := json.Marshal(map[string]any{"catalog_index": 0, "final_status": "FULL_PASS"})
			if err := os.WriteFile(filepath.Join(dir, "0000.json"), b, 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "future_timestamp", mutate: func(t *testing.T, r *Result, key []byte) {
			r.StartedAt = now.Add(time.Hour).Format(time.RFC3339Nano)
			r.FinishedAt = now.Add(2 * time.Hour).Format(time.RFC3339Nano)
			resign(t, r, key)
		}},
		{name: "manipulated_test_run_id_after_signing", mutate: func(t *testing.T, r *Result, key []byte) { r.TestRunID = "tampered-after-signing" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := baseManifest
			dir := t.TempDir()
			r := makeResult(t, m, entries[0], key, now)
			if tc.raw != nil {
				tc.raw(t, dir, r)
			} else {
				tc.mutate(t, &r, key)
				writeResult(t, dir, "0000.json", r)
			}
			assertRejected(t, dir, m, entries, now, key)
		})
	}
}

func TestDuplicateAndCopiedEvidenceRejected(t *testing.T) {
	m, entries, key, now := fixture(t)
	dir := t.TempDir()
	r := makeResult(t, m, entries[0], key, now)
	writeResult(t, dir, "0000.json", r)
	writeResult(t, dir, "copy.json", r)
	rs := assertRejected(t, dir, m, entries, now, key)
	if rs.Duplicates == 0 {
		t.Fatalf("copy was not classified as duplicate: %+v", rs)
	}
}

func TestConflictingPassFailEvidenceRejected(t *testing.T) {
	m, entries, key, now := fixture(t)
	dir := t.TempDir()
	pass := makeResult(t, m, entries[0], key, now)
	fail := pass
	fail.FinalStatus = "UNINSTALL_FAIL"
	fail.UninstallVerified = false
	fail.FailureStage = "UNINSTALL"
	fail.Failure = "synthetic conflict"
	resign(t, &fail, key)
	writeResult(t, dir, "pass.json", pass)
	writeResult(t, dir, "fail.json", fail)
	rs := assertRejected(t, dir, m, entries, now, key)
	if rs.Conflicts == 0 {
		t.Fatalf("conflict was not classified: %+v", rs)
	}
}

func TestValidEvidencePositiveFixture(t *testing.T) {
	m, entries, key, now := fixture(t)
	dir := t.TempDir()
	r := makeResult(t, m, entries[0], key, now)
	writeResult(t, dir, "0000.json", r)
	rs := LoadAndValidateResults(dir, m, entries, now, key)
	if len(rs.Valid) != 1 || len(rs.Issues) != 0 {
		t.Fatalf("positive fixture rejected: %+v", rs)
	}
}
