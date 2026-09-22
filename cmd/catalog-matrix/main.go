// catalog-matrix creates coverage inventory, never physical PASS evidence.
package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/linuxcatalog"
)

type row struct {
	ID        string `json:"app_id"`
	Name      string `json:"app_name"`
	Platform  string `json:"platform"`
	Version   string `json:"version"`
	Download  string `json:"download_test"`
	Install   string `json:"install_test"`
	Detection string `json:"detection_test"`
	Uninstall string `json:"uninstall_test"`
	Post      string `json:"post_uninstall_test"`
	Result    string `json:"result"`
	Reason    string `json:"failure_reason"`
	Evidence  string `json:"evidence"`
}

func main() {
	out := flag.String("out", "release/catalog-matrix", "output directory")
	win := flag.String("windows-blocker", "", "observed environment blocker; empty means unexecuted FAIL")
	lin := flag.String("linux-blocker", "", "observed environment blocker; empty means unexecuted FAIL")
	flag.Parse()
	var rows []row
	add := func(name, platform, reason, blocker string) {
		state := "FAIL"
		if reason == "" {
			reason = "Required lifecycle has not been executed"
			if blocker != "" {
				state = "BLOCKED_ENVIRONMENT"
				reason = blocker
			}
		}
		digest := sha256.Sum256([]byte(platform + "\x00" + name))
		rows = append(rows, row{hex.EncodeToString(digest[:]), name, platform, "", state, state, state, state, state, state, reason, ""})
	}
	for _, e := range catalog.BuildAuditEntries() {
		reason := ""
		if !e.LicensePolicyOK {
			reason = "Catalog license policy unresolved: " + string(e.LicenseClass)
		}
		if e.SystemComponent {
			reason = "System component lifecycle unsupported"
		}
		if e.UninstallStrategy == catalog.StrategyManualOnly {
			reason = "Automatic lifecycle unsupported"
		}
		add(e.Name, "windows", reason, *win)
	}
	for _, e := range linuxcatalog.Candidates {
		reason := ""
		if len(e.Apt)+len(e.Dnf)+len(e.Pacman)+len(e.Zypper)+len(e.Flatpak) == 0 {
			reason = "No explicit supported package mapping"
		}
		add(e.Name, "linux", reason, *lin)
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		panic(err)
	}
	b, err := json.MarshalIndent(struct {
		GeneratedAt string `json:"generated_at"`
		Kind        string `json:"kind"`
		Rows        []row  `json:"rows"`
	}{time.Now().UTC().Format(time.RFC3339), "coverage inventory; not physical evidence", rows}, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(filepath.Join(*out, "matrix.json"), append(b, '\n'), 0644); err != nil {
		panic(err)
	}
	f, err := os.Create(filepath.Join(*out, "matrix.csv"))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err = w.Write([]string{"app_id", "app_name", "platform", "version", "download_test", "install_test", "detection_test", "uninstall_test", "post_uninstall_test", "result", "failure_reason", "evidence"}); err != nil {
		panic(err)
	}
	for _, r := range rows {
		if err = w.Write([]string{r.ID, r.Name, r.Platform, r.Version, r.Download, r.Install, r.Detection, r.Uninstall, r.Post, r.Result, r.Reason, r.Evidence}); err != nil {
			panic(err)
		}
	}
	w.Flush()
	if err = w.Error(); err != nil {
		panic(err)
	}
	fmt.Printf("matrix rows=%d; physical PASS=0\n", len(rows))
}
