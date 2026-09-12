package main

// Shared by the UI installer and the physical test harness.
func packageInstallArgs(id, source string) []string {
	return []string{"install", "--id", id, "--exact", "--source", source, "--silent", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
}
