//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
)

func nsisRegisteredCommandLine(exe, registered string) (string, error) {
	at := strings.Index(registered, " _?=")
	if at < 0 {
		return "", fmt.Errorf("missing registered NSIS directory tail")
	}
	path := strings.TrimSpace(registered[at+4:])
	if !filepath.IsAbs(path) || strings.ContainsAny(path, "\"\r\n\x00") {
		return "", fmt.Errorf("invalid registered NSIS directory tail")
	}
	_, args, err := splitRegisteredCommandRaw(registered[:at])
	if err != nil {
		return "", err
	}
	line := syscall.EscapeArg(exe) + " /S"
	for _, arg := range args {
		if arg != "/S" {
			line += " " + syscall.EscapeArg(arg)
		}
	}
	return line + " _?=" + path, nil
}
