package releaseproof

import (
	"encoding/json"
	"fmt"
)

// ValidatePhysicalDocument is also used by the full release gate: authenticated
// booleans alone cannot establish a physical lifecycle.
func ValidatePhysicalDocument(b []byte, key []byte) error {
	if err := VerifyDocument(b, key); err != nil {
		return err
	}
	var r struct {
		DownloadProof struct {
			URL      string `json:"resolved_download_url"`
			HTTP     int    `json:"download_http_status"`
			Bytes    int64  `json:"downloaded_bytes"`
			SHA      string `json:"sha256"`
			Expected string `json:"expected_sha256"`
			Valid    bool   `json:"file_validation"`
		} `json:"download_proof"`
		FilesystemProof struct {
			Key             string   `json:"registry_key"`
			Paths           []string `json:"binary_paths"`
			Present         bool     `json:"registry_present"`
			BinariesPresent bool     `json:"binaries_present"`
			Removed         bool     `json:"registry_removed"`
			BinariesRemoved bool     `json:"binaries_removed"`
		} `json:"filesystem_proof"`
		FinalStatus        string `json:"final_status"`
		Executor           string `json:"executor"`
		Precheck           string `json:"precheck"`
		Commit             string `json:"git_commit"`
		VMID               string `json:"machine_id"`
		ID                 string `json:"resolved_id"`
		DownloadOK         bool   `json:"download_ok"`
		Downloaded         bool   `json:"download_artifact_present"`
		DownloadCode       int    `json:"download_exit_code"`
		InstallCode        int    `json:"install_exit_code"`
		UninstallCode      int    `json:"uninstall_exit_code"`
		InstallOK          bool   `json:"install_ok"`
		UninstallOK        bool   `json:"uninstall_ok"`
		UninstallAttempted bool   `json:"uninstall_attempted"`
		Reboot             bool   `json:"reboot_required"`
		Before             struct {
			ID       string `json:"id"`
			State    string `json:"state"`
			ExitCode int    `json:"exit_code"`
		} `json:"independent_before"`
		Installed struct {
			ID       string `json:"id"`
			State    string `json:"state"`
			ExitCode int    `json:"exit_code"`
		} `json:"independent_installed"`
		Removed struct {
			ID       string `json:"id"`
			State    string `json:"state"`
			ExitCode int    `json:"exit_code"`
		} `json:"independent_removed"`
		Environment struct {
			Status string `json:"status"`
			VMID   string `json:"vm_id"`
			Commit string `json:"git_commit"`
			Runner string `json:"runner_environment"`
			OS     string `json:"windows_version"`
			Arch   string `json:"architecture"`
		} `json:"environment"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}
	if r.FinalStatus != "FULL_PASS" {
		return nil
	}
	if r.DownloadProof.URL == "" || r.DownloadProof.HTTP != 200 || r.DownloadProof.Bytes <= 0 || len(r.DownloadProof.SHA) != 64 || r.DownloadProof.SHA != r.DownloadProof.Expected || !r.DownloadProof.Valid {
		return fmt.Errorf("verified physical HTTP download evidence missing")
	}
	if r.FilesystemProof.Key == "" || len(r.FilesystemProof.Paths) == 0 || !r.FilesystemProof.Present || !r.FilesystemProof.BinariesPresent || !r.FilesystemProof.Removed || !r.FilesystemProof.BinariesRemoved {
		return fmt.Errorf("independent registry/binary lifecycle evidence missing")
	}
	if r.Executor != "github-actions/windows" || r.Environment.Status != "READY" || r.Environment.Runner != "github-hosted" || r.Environment.OS == "" || r.Environment.Arch != "X64" || r.Environment.VMID != r.VMID || r.Environment.Commit != r.Commit {
		return fmt.Errorf("FULL_PASS requires matching hosted Windows Actions environment")
	}
	if r.Precheck != "CLEAN" || r.ID == "" || !r.DownloadOK || !r.Downloaded || !r.InstallOK || !r.UninstallOK || !r.UninstallAttempted || r.Reboot || r.DownloadCode != 0 || r.InstallCode != 0 || r.UninstallCode != 0 {
		return fmt.Errorf("incomplete physical lifecycle")
	}
	if r.Before.ID != r.ID || r.Installed.ID != r.ID || r.Removed.ID != r.ID || r.Before.State != "absent" || r.Installed.State != "present" || r.Removed.State != "absent" || uint32(r.Before.ExitCode) != 0x8a150014 || r.Installed.ExitCode != 0 || uint32(r.Removed.ExitCode) != 0x8a150014 {
		return fmt.Errorf("independent exact-ID lifecycle not proven")
	}
	return nil
}
