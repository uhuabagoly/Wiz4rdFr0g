package releaseproof

import (
	"encoding/json"
	"fmt"
)

// ValidatePhysicalDocument is also used by the full release gate: authenticated
// booleans alone cannot establish a physical lifecycle.
func ValidatePhysicalDocument(b []byte, key []byte) error {
	if err := VerifyDocument(b, key); err != nil { return err }
	var r struct {
		FinalStatus string `json:"final_status"`
		Executor string `json:"executor"`
		Precheck string `json:"precheck"`
		Commit string `json:"git_commit"`
		VMID string `json:"machine_id"`
		ID string `json:"resolved_id"`
		DownloadOK bool `json:"download_ok"`
		Downloaded bool `json:"download_artifact_present"`
		DownloadCode int `json:"download_exit_code"`
		InstallCode int `json:"install_exit_code"`
		UninstallCode int `json:"uninstall_exit_code"`
		InstallOK bool `json:"install_ok"`
		UninstallOK bool `json:"uninstall_ok"`
		UninstallAttempted bool `json:"uninstall_attempted"`
		Reboot bool `json:"reboot_required"`
		Before struct { ID string `json:"id"`; State string `json:"state"`; ExitCode int `json:"exit_code"` } `json:"independent_before"`
		Installed struct { ID string `json:"id"`; State string `json:"state"`; ExitCode int `json:"exit_code"` } `json:"independent_installed"`
		Removed struct { ID string `json:"id"`; State string `json:"state"`; ExitCode int `json:"exit_code"` } `json:"independent_removed"`
		Environment struct { Status string `json:"status"`; VMID string `json:"vm_id"`; Commit string `json:"git_commit"`; Runner string `json:"runner_environment"`; OS string `json:"windows_version"`; Arch string `json:"architecture"` } `json:"environment"`
	}
	if err := json.Unmarshal(b, &r); err != nil { return err }
	if r.FinalStatus != "FULL_PASS" { return nil }
	if r.Executor != "github-actions/windows" || r.Environment.Status != "READY" || r.Environment.Runner != "github-hosted" || r.Environment.OS == "" || r.Environment.Arch != "X64" || r.Environment.VMID != r.VMID || r.Environment.Commit != r.Commit { return fmt.Errorf("FULL_PASS requires matching hosted Windows Actions environment") }
	if r.Precheck != "CLEAN" || r.ID == "" || !r.DownloadOK || !r.Downloaded || !r.InstallOK || !r.UninstallOK || !r.UninstallAttempted || r.Reboot || r.DownloadCode != 0 || r.InstallCode != 0 || r.UninstallCode != 0 { return fmt.Errorf("incomplete physical lifecycle") }
	if r.Before.ID != r.ID || r.Installed.ID != r.ID || r.Removed.ID != r.ID || r.Before.State != "absent" || r.Installed.State != "present" || r.Removed.State != "absent" || uint32(r.Before.ExitCode) != 0x8a150014 || r.Installed.ExitCode != 0 || uint32(r.Removed.ExitCode) != 0x8a150014 { return fmt.Errorf("independent exact-ID lifecycle not proven") }
	return nil
}
