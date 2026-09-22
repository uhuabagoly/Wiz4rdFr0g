package linuxpkg

import (
	"fmt"
	"strings"
)

// APTCandidate respects repository pin priorities. Lexical maximum versions
// can select backports whose matching dependencies are not APT candidates.
func APTCandidate(policy string) (string, error) {
	for _, line := range strings.Split(policy, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Candidate:") {
			version := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Candidate:"))
			if version != "" && version != "(none)" {
				return version, nil
			}
		}
	}
	return "", fmt.Errorf("APT has no installable candidate")
}

// Exact absent states only; a query error or partial removal is never absence.
func DPKGRemoved(output string, exitCode int) bool {
	text := strings.TrimSpace(output)
	if exitCode == 1 {
		return strings.Contains(text, "no packages found matching")
	}
	if exitCode != 0 {
		return false
	}
	return text == "deinstall ok config-files" || text == "unknown ok not-installed" || text == "deinstall ok not-installed" || text == "purge ok not-installed"
}
