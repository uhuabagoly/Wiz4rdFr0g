package releaseproof

import (
	"encoding/json"
	"regexp"
	"strings"
)

func (o installedObservation) provesLICEcap(want string) bool {
	if o.ID != "Cockos.LICEcap" || o.ExitCode != 0 || o.State != want || len(o.Command) != 5 || strings.Join(o.Command[:4], "\x00") != "powershell.exe\x00-NoProfile\x00-NonInteractive\x00-Command" || o.Command[4] != LICEcapIndependentScript {
		return false
	}
	var p struct {
		State    string  `json:"state"`
		Registry string  `json:"registry"`
		View     int     `json:"view"`
		Root     *string `json:"registered_root"`
		Files    []struct {
			Path  string `json:"path"`
			Bytes int64  `json:"bytes"`
			PE    bool   `json:"pe"`
		} `json:"files"`
	}
	if json.Unmarshal([]byte(o.Output), &p) != nil || p.State != want || p.Registry != `HKEY_LOCAL_MACHINE\Software\LICEcap` || p.View != 32 {
		return false
	}
	if want == "absent" {
		return p.Root == nil && len(p.Files) == 0
	}
	if want != "present" || p.Root == nil || !regexp.MustCompile(`(?i)^[a-z]:\\Program Files \(x86\)\\LICEcap\\?$`).MatchString(*p.Root) || len(p.Files) != 2 {
		return false
	}
	root := strings.TrimRight(*p.Root, `\`)
	wantPaths := map[string]bool{strings.ToLower(root + `\LICEcap.exe`): true, strings.ToLower(root + `\Uninstall.exe`): true}
	for _, f := range p.Files {
		key := strings.ToLower(f.Path)
		if !wantPaths[key] || f.Bytes <= 0 || !f.PE {
			return false
		}
		delete(wantPaths, key)
	}
	return len(wantPaths) == 0
}
