package releasegate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

type Result struct {
	releaseproof.EvidenceStatement
	Signature    string `json:"signature"`
	Name         string `json:"name"`
	FailureStage string `json:"failure_stage"`
	Failure      string `json:"failure"`
	SkipReason   string `json:"skip_reason"`
}

type Issue struct {
	Path         string `json:"path"`
	CatalogIndex int    `json:"catalog_index,omitempty"`
	Code         string `json:"code"`
	Message      string `json:"message"`
}

type ResultSet struct {
	Files      int
	Valid      map[int]Result
	Issues     []Issue
	Invalid    int
	Mismatched int
	Duplicates int
	Conflicts  int
}

var ignoredJSONNames = map[string]bool{
	"summary.json":              true,
	"execution_summary.json":    true,
	"environment_evidence.json": true,
	"physical_test_plan.json":   true,
	"batches.json":              true,
}

var requiredFields = []string{
	"schema_version", "build_id", "app_version", "git_commit", "catalog_fingerprint", "artifact_sha256",
	"test_run_id", "catalog_index", "catalog_app_id", "catalog_app_name", "started_at", "finished_at",
	"machine_id", "final_status", "install_verified", "uninstall_verified", "duration_seconds", "signature",
}

func LoadManifest(path string) (releaseproof.BuildManifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return releaseproof.BuildManifest{}, err
	}
	var m releaseproof.BuildManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return releaseproof.BuildManifest{}, err
	}
	return m, nil
}

func ValidateArtifactFiles(m releaseproof.BuildManifest, projectRoot string) []Issue {
	var issues []Issue
	required := []string{releaseproof.PrimaryWindowsArtifactKey, releaseproof.WindowsSetupArtifactKey, releaseproof.InstallerPayloadArtifactKey, releaseproof.LinuxAppArtifactKey, releaseproof.LinuxPackageArtifactKey}
	for _, key := range required {
		art, ok := m.Artifacts[key]
		if !ok {
			issues = append(issues, Issue{Code: "missing_artifact", Message: "build manifest missing artifact " + key})
			continue
		}
		rawPath := filepath.FromSlash(strings.TrimSpace(art.Path))
		cleanPath := filepath.Clean(rawPath)
		if rawPath == "" || filepath.IsAbs(rawPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
			issues = append(issues, Issue{Path: art.Path, Code: "unsafe_artifact_path", Message: "artifact path must be relative and remain inside the project root"})
			continue
		}
		path := filepath.Join(projectRoot, cleanPath)
		hash, size, err := releaseproof.FileSHA256(path)
		if err != nil {
			issues = append(issues, Issue{Path: path, Code: "artifact_unreadable", Message: err.Error()})
			continue
		}
		if !strings.EqualFold(hash, art.SHA256) || size != art.Size {
			issues = append(issues, Issue{Path: path, Code: "artifact_manifest_mismatch", Message: fmt.Sprintf("%s hash/size differs from build manifest", key)})
		}
	}
	if a, okA := m.Artifacts[releaseproof.PrimaryWindowsArtifactKey]; okA {
		if p, okP := m.Artifacts[releaseproof.InstallerPayloadArtifactKey]; okP && (!strings.EqualFold(a.SHA256, p.SHA256) || a.Size != p.Size) {
			issues = append(issues, Issue{Code: "payload_mismatch", Message: "installer payload does not match Windows application artifact"})
		}
	}
	sortIssues(issues)
	return issues
}

func LoadAndValidateResults(dir string, manifest releaseproof.BuildManifest, entries []catalogpkg.AuditEntry, now time.Time, key []byte) ResultSet {
	rs := ResultSet{Valid: map[int]Result{}}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(files)
	candidates := map[int][]struct {
		path string
		r    Result
	}{}
	appIDOwners := map[string][]int{}
	for _, path := range files {
		if ignoredJSONNames[strings.ToLower(filepath.Base(path))] {
			continue
		}
		rs.Files++
		b, err := os.ReadFile(path)
		if err != nil {
			rs.addInvalid(Issue{Path: path, Code: "read_error", Message: err.Error()})
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			rs.addInvalid(Issue{Path: path, Code: "corrupt_json", Message: err.Error()})
			continue
		}
		missing := []string{}
		for _, f := range requiredFields {
			if _, ok := raw[f]; !ok {
				missing = append(missing, f)
			}
		}
		if len(missing) != 0 {
			rs.addInvalid(Issue{Path: path, Code: "missing_required_fields", Message: "missing: " + strings.Join(missing, ", ")})
			continue
		}
		var r Result
		if err := json.Unmarshal(b, &r); err != nil {
			rs.addInvalid(Issue{Path: path, Code: "decode_error", Message: err.Error()})
			continue
		}
		if r.CatalogIndex < 0 || r.CatalogIndex >= len(entries) {
			rs.addInvalid(Issue{Path: path, CatalogIndex: r.CatalogIndex, Code: "catalog_index_out_of_range", Message: "catalog_index does not identify a current catalog entry"})
			continue
		}
		vissues := releaseproof.ValidateEvidence(r.EvidenceStatement, r.Signature, manifest, entries[r.CatalogIndex], now, key)
		if err := releaseproof.ValidatePhysicalDocument(b, key); err != nil {
			vissues = append(vissues, releaseproof.ValidationIssue{Code: "physical_document_invalid", Message: err.Error()})
		}
		if len(vissues) != 0 {
			for _, vi := range vissues {
				issue := Issue{Path: path, CatalogIndex: r.CatalogIndex, Code: vi.Code, Message: vi.Message}
				if isMismatchCode(vi.Code) {
					rs.addMismatch(issue)
				} else {
					rs.addInvalid(issue)
				}
			}
			continue
		}
		candidates[r.CatalogIndex] = append(candidates[r.CatalogIndex], struct {
			path string
			r    Result
		}{path, r})
		appIDOwners[r.CatalogAppID] = append(appIDOwners[r.CatalogAppID], r.CatalogIndex)
	}

	// A catalog entry must have exactly one evidence file. Even identical copies are rejected.
	for idx, list := range candidates {
		if len(list) == 1 {
			rs.Valid[idx] = list[0].r
			continue
		}
		conflict := false
		first := list[0].r.EvidenceStatement
		for _, item := range list[1:] {
			if item.r.EvidenceStatement != first {
				conflict = true
				break
			}
		}
		for _, item := range list {
			rs.Issues = append(rs.Issues, Issue{Path: item.path, CatalogIndex: idx, Code: "duplicate_evidence", Message: "multiple evidence files exist for one catalog entry"})
		}
		rs.Duplicates += len(list)
		if conflict {
			rs.Conflicts++
			rs.Issues = append(rs.Issues, Issue{CatalogIndex: idx, Code: "conflicting_evidence", Message: "duplicate evidence files contain conflicting signed statements"})
		}
	}

	// CatalogAppID is designed to be unique. Reuse across different indexes is evidence copying/collision.
	for appID, indexes := range appIDOwners {
		uniq := map[int]bool{}
		for _, i := range indexes {
			uniq[i] = true
		}
		if len(uniq) <= 1 {
			continue
		}
		for idx := range uniq {
			delete(rs.Valid, idx)
			rs.Issues = append(rs.Issues, Issue{CatalogIndex: idx, Code: "catalog_app_id_collision", Message: "catalog_app_id reused across catalog entries: " + appID})
			rs.Conflicts++
		}
	}
	sortIssues(rs.Issues)
	return rs
}

func (rs *ResultSet) addInvalid(i Issue)  { rs.Invalid++; rs.Issues = append(rs.Issues, i) }
func (rs *ResultSet) addMismatch(i Issue) { rs.Mismatched++; rs.Issues = append(rs.Issues, i) }
func isMismatchCode(code string) bool {
	switch code {
	case "build_id_mismatch", "app_version_mismatch", "git_commit_mismatch", "catalog_fingerprint_mismatch", "artifact_hash_mismatch", "catalog_index_mismatch", "catalog_app_id_mismatch", "catalog_app_name_mismatch", "schema_mismatch":
		return true
	default:
		return false
	}
}
func sortIssues(v []Issue) {
	sort.Slice(v, func(i, j int) bool {
		if v[i].CatalogIndex != v[j].CatalogIndex {
			return v[i].CatalogIndex < v[j].CatalogIndex
		}
		if v[i].Code != v[j].Code {
			return v[i].Code < v[j].Code
		}
		return v[i].Path < v[j].Path
	})
}
