package releaseproof

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type installedObservation struct {
	Command           []string `json:"command"`
	Output            string   `json:"output"`
	ID                string   `json:"id"`
	State             string   `json:"state"`
	ExitCode          int      `json:"exit_code"`
	InventoryCommand  []string `json:"inventory_command"`
	InventoryExitCode *int     `json:"inventory_exit_code"`
	InventoryOutput   string   `json:"inventory_output"`
}

func (o installedObservation) provesPresent() bool {
	if o.State != "present" || o.ID == "" {
		return false
	}
	if o.ID == "Cockos.LICEcap" {
		return o.provesLICEcap("present")
	}
	if o.ExitCode == 0 {
		return true
	}
	// A known exact-query false negative is acceptable only when the signed
	// successful unfiltered inventory contains this complete package ID.
	return uint32(o.ExitCode) == 0x8a150014 && o.InventoryExitCode != nil && *o.InventoryExitCode == 0 &&
		strings.Join(o.InventoryCommand, "\x00") == "winget.exe\x00list\x00--accept-source-agreements\x00--disable-interactivity" &&
		regexp.MustCompile(`(?m)(?:^|\s)`+regexp.QuoteMeta(o.ID)+`(?:\s|$)`).MatchString(o.InventoryOutput)
}

func (o installedObservation) provesAbsent() bool {
	if o.ID == "Cockos.LICEcap" {
		return o.provesLICEcap("absent")
	}
	return o.ID != "" && o.State == "absent" && uint32(o.ExitCode) == 0x8a150014
}

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
			SeafileRetention *SeafileRetentionProof `json:"seafile_retention"`
			PreservedData    []struct {
				Path      string `json:"path"`
				Bytes     int64  `json:"bytes"`
				SHA256    string `json:"sha256"`
				Preserved bool   `json:"preserved"`
			} `json:"preserved_data"`
			Appx            *AppxProof `json:"appx"`
			Key             string     `json:"registry_key"`
			Paths           []string   `json:"binary_paths"`
			Present         bool       `json:"registry_present"`
			BinariesPresent bool       `json:"binaries_present"`
			Removed         bool       `json:"registry_removed"`
			BinariesRemoved bool       `json:"binaries_removed"`
		} `json:"filesystem_proof"`
		FinalStatus        string               `json:"final_status"`
		Executor           string               `json:"executor"`
		Precheck           string               `json:"precheck"`
		Commit             string               `json:"git_commit"`
		VMID               string               `json:"machine_id"`
		ID                 string               `json:"resolved_id"`
		DownloadOK         bool                 `json:"download_ok"`
		Downloaded         bool                 `json:"download_artifact_present"`
		DownloadCode       int                  `json:"download_exit_code"`
		InstallCode        int                  `json:"install_exit_code"`
		UninstallCode      int                  `json:"uninstall_exit_code"`
		InstallOK          bool                 `json:"install_ok"`
		UninstallOK        bool                 `json:"uninstall_ok"`
		UninstallAttempted bool                 `json:"uninstall_attempted"`
		Reboot             bool                 `json:"reboot_required"`
		Before             installedObservation `json:"independent_before"`
		Installed          installedObservation `json:"independent_installed"`
		Removed            installedObservation `json:"independent_removed"`
		Environment        struct {
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
	registrationProven := r.FilesystemProof.Key != "" && r.FilesystemProof.Present && r.FilesystemProof.Removed
	if a := r.FilesystemProof.Appx; a != nil {
		registrationProven = validAppxLifecycle(r.ID, a, r.FilesystemProof.Paths)
	}
	if !registrationProven || len(r.FilesystemProof.Paths) == 0 || !r.FilesystemProof.BinariesPresent || !r.FilesystemProof.BinariesRemoved {
		return fmt.Errorf("independent registry/binary lifecycle evidence missing")
	}
	if r.ID == "Canonical.Multipass" {
		p := r.FilesystemProof.PreservedData
		if len(p) != 1 || !p[0].Preserved || p[0].Bytes <= 0 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(p[0].SHA256) || !regexp.MustCompile(`(?i)^[a-z]:\\ProgramData\\Multipass\\Wiz4rdFr0g-retention-[a-z0-9]+\.txt$`).MatchString(p[0].Path) {
			return fmt.Errorf("Multipass data-retention witness missing or altered")
		}
	}
	if r.ID == "Seafile.Seafile" && !r.FilesystemProof.SeafileRetention.valid() {
		return fmt.Errorf("Seafile settings-retention witness missing or altered")
	}
	if r.Executor != "github-actions/windows" || r.Environment.Status != "READY" || r.Environment.Runner != "github-hosted" || r.Environment.OS == "" || r.Environment.Arch != "X64" || r.Environment.VMID != r.VMID || r.Environment.Commit != r.Commit {
		return fmt.Errorf("FULL_PASS requires matching hosted Windows Actions environment")
	}
	if r.Precheck != "CLEAN" || r.ID == "" || !r.DownloadOK || !r.Downloaded || !r.InstallOK || !r.UninstallOK || !r.UninstallAttempted || r.Reboot || r.DownloadCode != 0 || r.InstallCode != 0 || r.UninstallCode != 0 {
		return fmt.Errorf("incomplete physical lifecycle")
	}
	if r.Before.ID != r.ID || r.Installed.ID != r.ID || r.Removed.ID != r.ID || !r.Before.provesAbsent() || !r.Installed.provesPresent() || !r.Removed.provesAbsent() {
		return fmt.Errorf("independent exact-ID lifecycle not proven")
	}
	return nil
}
