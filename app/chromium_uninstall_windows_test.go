//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChromiumUninstallerUsesChildOnlySameVolumeTemp(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "Application")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	originalTemp := os.Getenv("TEMP")
	cmd := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	code, out, err := runChromiumVendorUninstaller(context.Background(), cmd, []string{"/D", "/C", "echo %TEMP%&echo %TMP%&cd"}, root)
	if err != nil || code != 0 {
		t.Fatal(code, out, err)
	}
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(out, "\r", "")), "\n")
	if len(lines) != 3 || lines[0] != lines[1] || !strings.EqualFold(lines[0], lines[2]) || filepath.Dir(lines[0]) != parent {
		t.Fatalf("child TEMP, TMP and cwd must be the same sibling of Application: %q", out)
	}
	if os.Getenv("TEMP") != originalTemp {
		t.Fatal("parent environment changed")
	}
	if _, err := os.Stat(lines[0]); !os.IsNotExist(err) {
		t.Fatal("owned empty temporary directory was not cleaned up", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("application directory was modified", err)
	}
}
