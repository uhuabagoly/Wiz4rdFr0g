package releaseproof

import "strings"

// Publisher-bound package families observed in physical run 36304661281.
func ExpectedAppxFamily(id string) string {
	switch id {
	case "FilesCommunity.Files":
		// Stable 4.2.9.0 publisher CDN package, exact WinGet installer manifest.
		return "Files_1y0xx7n9077q4"
	case "Proton.ProtonPass":
		return "ProtonPass_158qdr94jw63p"
	case "M2Team.NanaZip":
		return "40174MouriNaruto.NanaZip_gnj4mf6z9tkrc"
	}
	return ""
}

type AppxProof struct {
	Family          string   `json:"family"`
	FullName        string   `json:"full_name"`
	InstallLocation string   `json:"install_location"`
	Executables     []string `json:"executables"`
	Present         bool     `json:"present"`
	Removed         bool     `json:"removed"`
}

func validAppxLifecycle(id string, a *AppxProof, paths []string) bool {
	expected := ExpectedAppxFamily(id)
	if expected == "" || a.Family != expected || a.FullName == "" || a.InstallLocation == "" || !a.Present || !a.Removed || len(a.Executables) == 0 || len(paths) == 0 {
		return false
	}
	split := strings.LastIndex(expected, "_")
	if !strings.HasPrefix(a.FullName, expected[:split]+"_") || !strings.HasSuffix(a.FullName, "__"+expected[split+1:]) {
		return false
	}
	root := strings.TrimRight(strings.ReplaceAll(a.InstallLocation, "/", `\`), `\`)
	for _, path := range paths {
		found := false
		for _, exe := range a.Executables {
			exe = strings.ReplaceAll(exe, "/", `\`)
			if exe == "" || strings.Contains(exe, "..") || strings.HasPrefix(exe, `\`) || strings.Contains(exe, ":") {
				continue
			}
			if strings.EqualFold(strings.ReplaceAll(path, "/", `\`), root+`\`+exe) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
