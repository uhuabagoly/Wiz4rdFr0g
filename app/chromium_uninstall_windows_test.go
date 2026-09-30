//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChromiumVendorStatusDoesNotRelaxOtherUninstallers(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	root := filepath.Join(local, "Chromium", "Application")
	r := registryPackage{DisplayName: "Chromium", DisplayVersion: "154.0.8037.58", Scope: "user", InstallLocation: root, RegistryKey: `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Chromium`, UninstallString: `"` + filepath.Join(root, "154.0.8037.58", "Installer", "setup.exe") + `" --uninstall`}
	for _, tc := range []struct {
		code            int
		success, reboot bool
	}{{19, true, false}, {20, false, false}, {21, false, false}, {29, true, true}} {
		got := chromiumUninstallAttempt(appDef{Name: "Chromium"}, r, tc.code, "", fmt.Errorf("exit status %d", tc.code))
		if got.ExitCode != tc.code || got.Success != tc.success || got.RebootRequired != tc.reboot {
			t.Fatalf("status %d incorrectly classified: %+v", tc.code, got)
		}
	}
	if chromiumUninstallAttempt(appDef{Name: "Other"}, r, 19, "", nil).Success {
		t.Fatal("code 19 accepted for an unrelated application")
	}
	r.RegistryKey += "-unrelated"
	if chromiumUninstallAttempt(appDef{Name: "Chromium"}, r, 19, "", nil).Success {
		t.Fatal("code 19 accepted without the exact browser registration")
	}
}

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
