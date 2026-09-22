package linuxpkg

import (
	"regexp"
	"strings"
)

// Select only payload packages explicitly declared by the repository's
// metapackage. Never derive a package ID from an application display name.
func APTPayloadPackage(meta, dependencies string, available map[string]bool) string {
	server := regexp.MustCompile(`^postgresql-[0-9]+$`)
	for _, line := range strings.Split(dependencies, "\n") {
		fields := strings.Fields(strings.TrimLeft(strings.TrimSpace(line), "|"))
		if len(fields) != 2 || fields[0] != "Depends:" || !available[fields[1]] {
			continue
		}
		p := fields[1]
		if (meta == "postgresql" && server.MatchString(p)) || (meta == "emacs" && (p == "emacs-gtk" || p == "emacs-pgtk" || p == "emacs-lucid" || p == "emacs-nox")) || (meta == "qemu-system" && p == "qemu-system-x86") {
			return p
		}
	}
	return meta
}
