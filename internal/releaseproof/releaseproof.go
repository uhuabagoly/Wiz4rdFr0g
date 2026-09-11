package releaseproof

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

const (
	AppVersion                  = "0.6.23-release-gate"
	EvidenceSchemaVersion       = 2
	BuildManifestSchemaVersion  = 1
	EvidenceKeyEnvironment      = "WIZ4RDFR0G_EVIDENCE_HMAC_KEY"
	MinimumEvidenceKeyBytes     = 32
	PrimaryWindowsArtifactKey   = "windows_app"
	WindowsSetupArtifactKey     = "windows_setup"
	InstallerPayloadArtifactKey = "installer_payload"
	LinuxAppArtifactKey         = "linux_app"
	LinuxPackageArtifactKey     = "linux_package"
)

var gitCommitRE = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type BuildManifest struct {
	SchemaVersion         int                 `json:"schema_version"`
	GeneratedAt           string              `json:"generated_at"`
	AppVersion            string              `json:"app_version"`
	GitCommit             string              `json:"git_commit"`
	CatalogFingerprint    string              `json:"catalog_fingerprint"`
	EvidenceSchemaVersion int                 `json:"evidence_schema_version"`
	BuildID               string              `json:"build_id"`
	Artifacts             map[string]Artifact `json:"artifacts"`
	PayloadConsistent     bool                `json:"payload_consistent"`
}

type EvidenceStatement struct {
	SchemaVersion      int     `json:"schema_version"`
	BuildID            string  `json:"build_id"`
	AppVersion         string  `json:"app_version"`
	GitCommit          string  `json:"git_commit"`
	CatalogFingerprint string  `json:"catalog_fingerprint"`
	ArtifactSHA256     string  `json:"artifact_sha256"`
	TestRunID          string  `json:"test_run_id"`
	CatalogIndex       int     `json:"catalog_index"`
	CatalogAppID       string  `json:"catalog_app_id"`
	CatalogAppName     string  `json:"catalog_app_name"`
	StartedAt          string  `json:"started_at"`
	FinishedAt         string  `json:"finished_at"`
	MachineID          string  `json:"machine_id"`
	FinalStatus        string  `json:"final_status"`
	FailureStage       string  `json:"failure_stage,omitempty"`
	Failure            string  `json:"failure,omitempty"`
	SkipReason         string  `json:"skip_reason,omitempty"`
	InstallVerified    bool    `json:"install_verified"`
	UninstallVerified  bool    `json:"uninstall_verified"`
	DurationSeconds    float64 `json:"duration_seconds"`
}

type ValidationIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func CatalogFingerprint(entries []catalogpkg.AuditEntry) (string, error) {
	// BuildAuditEntries is deterministic; encoding a slice of structs preserves field and item order.
	b, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func CurrentCatalogFingerprint() (string, error) {
	return CatalogFingerprint(catalogpkg.BuildAuditEntries())
}

func CatalogAppID(entry catalogpkg.AuditEntry) string {
	// This is a catalog-entry identity, not merely a package-manager ID. Aliases can
	// legitimately share a Winget ID, so include the stable catalog semantics too.
	raw := strings.Join([]string{
		strings.ToLower(strings.TrimSpace(entry.Name)),
		strings.ToLower(strings.TrimSpace(entry.Category)),
		strings.ToLower(strings.TrimSpace(entry.CanonicalName)),
		strings.ToLower(strings.TrimSpace(entry.CatalogAliasOf)),
		strings.ToLower(strings.TrimSpace(entry.WingetID)),
	}, "\x00")
	sum := sha256.Sum256([]byte(raw))
	return "catalog-entry-sha256:" + hex.EncodeToString(sum[:])
}

func BuildID(appVersion, gitCommit, catalogFingerprint, artifactSHA256 string, evidenceSchemaVersion int) string {
	raw := strings.Join([]string{
		strings.TrimSpace(appVersion),
		strings.ToLower(strings.TrimSpace(gitCommit)),
		strings.ToLower(strings.TrimSpace(catalogFingerprint)),
		strings.ToLower(strings.TrimSpace(artifactSHA256)),
		fmt.Sprintf("evidence-schema:%d", evidenceSchemaVersion),
	}, "\n")
	sum := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func FileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func ValidGitCommit(s string) bool {
	return gitCommitRE.MatchString(strings.TrimSpace(s))
}

func ValidateEvidenceKey(key []byte) error {
	if len(key) < MinimumEvidenceKeyBytes {
		return fmt.Errorf("evidence HMAC key must contain at least %d bytes", MinimumEvidenceKeyBytes)
	}
	return nil
}

func EvidenceSignature(statement EvidenceStatement, key []byte) (string, error) {
	if err := ValidateEvidenceKey(key); err != nil {
		return "", err
	}
	b, err := json.Marshal(statement)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(b)
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil)), nil
}

func VerifyEvidenceSignature(statement EvidenceStatement, signature string, key []byte) error {
	expected, err := EvidenceSignature(statement, key)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature))) {
		return errors.New("evidence signature mismatch")
	}
	return nil
}

func ValidateManifest(m BuildManifest, currentCatalogFingerprint string) []ValidationIssue {
	var issues []ValidationIssue
	add := func(code, msg string) { issues = append(issues, ValidationIssue{Code: code, Message: msg}) }
	if m.SchemaVersion != BuildManifestSchemaVersion {
		add("manifest_schema_mismatch", fmt.Sprintf("manifest schema=%d expected=%d", m.SchemaVersion, BuildManifestSchemaVersion))
	}
	if m.EvidenceSchemaVersion != EvidenceSchemaVersion {
		add("evidence_schema_mismatch", fmt.Sprintf("manifest evidence schema=%d expected=%d", m.EvidenceSchemaVersion, EvidenceSchemaVersion))
	}
	if m.AppVersion != AppVersion {
		add("app_version_mismatch", fmt.Sprintf("manifest app_version=%q expected=%q", m.AppVersion, AppVersion))
	}
	if !ValidGitCommit(m.GitCommit) {
		add("invalid_git_commit", "manifest git_commit is not a 40-hex Git commit SHA")
	}
	if strings.TrimSpace(m.CatalogFingerprint) == "" || !strings.EqualFold(m.CatalogFingerprint, currentCatalogFingerprint) {
		add("catalog_fingerprint_mismatch", "manifest catalog fingerprint does not match the current catalog")
	}
	primary, ok := m.Artifacts[PrimaryWindowsArtifactKey]
	if !ok {
		add("missing_primary_artifact", "manifest is missing windows_app artifact")
	} else {
		expectedID := BuildID(m.AppVersion, m.GitCommit, m.CatalogFingerprint, primary.SHA256, m.EvidenceSchemaVersion)
		if m.BuildID != expectedID {
			add("build_id_mismatch", "manifest build_id does not match manifest identity fields")
		}
	}
	if !m.PayloadConsistent {
		add("payload_inconsistent", "installer payload was not proven equal to the Windows application artifact")
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Code == issues[j].Code {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}

func ValidateEvidence(statement EvidenceStatement, signature string, manifest BuildManifest, entry catalogpkg.AuditEntry, now time.Time, key []byte) []ValidationIssue {
	var issues []ValidationIssue
	add := func(code, msg string) { issues = append(issues, ValidationIssue{Code: code, Message: msg}) }

	if statement.SchemaVersion != EvidenceSchemaVersion {
		add("schema_mismatch", fmt.Sprintf("evidence schema=%d expected=%d", statement.SchemaVersion, EvidenceSchemaVersion))
	}
	if statement.BuildID != manifest.BuildID {
		add("build_id_mismatch", "evidence build_id does not match release build")
	}
	if statement.AppVersion != manifest.AppVersion {
		add("app_version_mismatch", "evidence app_version does not match release build")
	}
	if !strings.EqualFold(statement.GitCommit, manifest.GitCommit) {
		add("git_commit_mismatch", "evidence git_commit does not match release build")
	}
	if !strings.EqualFold(statement.CatalogFingerprint, manifest.CatalogFingerprint) {
		add("catalog_fingerprint_mismatch", "evidence catalog fingerprint does not match release build")
	}
	primary := manifest.Artifacts[PrimaryWindowsArtifactKey]
	if !strings.EqualFold(statement.ArtifactSHA256, primary.SHA256) {
		add("artifact_hash_mismatch", "evidence artifact SHA256 does not match release Windows application")
	}
	if statement.CatalogIndex != entry.Index {
		add("catalog_index_mismatch", fmt.Sprintf("evidence index=%d expected=%d", statement.CatalogIndex, entry.Index))
	}
	expectedAppID := CatalogAppID(entry)
	if statement.CatalogAppID != expectedAppID {
		add("catalog_app_id_mismatch", "evidence catalog_app_id does not match catalog entry")
	}
	if statement.CatalogAppName != entry.Name {
		add("catalog_app_name_mismatch", fmt.Sprintf("evidence name=%q expected=%q", statement.CatalogAppName, entry.Name))
	}
	if strings.TrimSpace(statement.TestRunID) == "" {
		add("missing_test_run_id", "evidence test_run_id is empty")
	}
	if strings.TrimSpace(statement.MachineID) == "" {
		add("missing_machine_id", "evidence machine_id is empty")
	}
	if strings.TrimSpace(statement.FinalStatus) == "" {
		add("missing_final_status", "evidence final_status is empty")
	}
	if statement.FinalStatus == "FULL_PASS" && (!statement.InstallVerified || !statement.UninstallVerified) {
		add("invalid_full_pass", "FULL_PASS requires install_verified=true and uninstall_verified=true")
	}
	if statement.DurationSeconds < 0 {
		add("negative_duration", "duration_seconds is negative")
	}
	started, startErr := time.Parse(time.RFC3339Nano, statement.StartedAt)
	finished, finishErr := time.Parse(time.RFC3339Nano, statement.FinishedAt)
	if startErr != nil {
		add("invalid_started_at", "started_at is not RFC3339/RFC3339Nano")
	}
	if finishErr != nil {
		add("invalid_finished_at", "finished_at is not RFC3339/RFC3339Nano")
	}
	if startErr == nil && finishErr == nil {
		if finished.Before(started) {
			add("negative_time_range", "finished_at precedes started_at")
		}
		// Allow only a small amount of clock skew between runner and gate.
		if started.After(now.Add(2*time.Minute)) || finished.After(now.Add(2*time.Minute)) {
			add("future_timestamp", "evidence timestamp is more than two minutes in the future")
		}
	}
	if err := VerifyEvidenceSignature(statement, signature, key); err != nil {
		add("signature_invalid", err.Error())
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Code == issues[j].Code {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}
