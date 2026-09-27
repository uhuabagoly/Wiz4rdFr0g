package main

import "strings"

// Installer caches and maintenance tools are not application payload proof.
func vmApplicationPayloadPath(path string) bool {
	path = strings.ToLower(strings.ReplaceAll(path, `/`, `\`))
	parts := strings.Split(path, `\`)
	for _, part := range parts {
		if part == "package cache" {
			return false
		}
	}
	name := parts[len(parts)-1]
	return !strings.Contains(name, "unins") && !strings.Contains(name, "update") && name != "squirrel.exe"
}
