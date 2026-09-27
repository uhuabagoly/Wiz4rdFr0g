//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const python313MachineKey = `HKEY_LOCAL_MACHINE\Software\Python\PythonCore\3.13\InstallPath`

func registeredPython313Machine(ctx context.Context) (string, error) {
	code, out, err := runDirectProcess(ctx, "reg.exe", []string{"query", python313MachineKey, "/v", "ExecutablePath", "/reg:64"})
	if code == 1 && strings.Contains(strings.ToLower(out), "unable to find") {
		return "", nil
	}
	m := regexp.MustCompile(`(?m)^\s*ExecutablePath\s+REG_SZ\s+(.+?)\s*$`).FindStringSubmatch(out)
	if err != nil || code != 0 || len(m) != 2 {
		return "", fmt.Errorf("cannot establish Python baseline: %s", compactLog(out))
	}
	return strings.TrimSpace(m[1]), nil
}

type vmPythonBaseline struct {
	Path     string            `json:"path"`
	Version  string            `json:"version"`
	Command  string            `json:"command"`
	ExitCode int               `json:"exit_code"`
	Output   string            `json:"output"`
	Removed  bool              `json:"removed"`
	Logs     map[string]string `json:"logs,omitempty"`
}

// Hosted images contain Python MSI features without a discoverable Burn bundle.
// Remove that exact matching baseline with the verified publisher installer,
// then require both its interpreter and PEP 514 registration to be gone.
func vmPreparePythonBaseline(ctx context.Context, version, workRoot, started string, proof vmDownloadProof) (*vmPythonBaseline, error) {
	path, err := registeredPython313Machine(ctx)
	if err != nil || path == "" {
		return nil, err
	}
	r := &vmPythonBaseline{Path: path, ExitCode: -999}
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || !proof.Valid || proof.Format != "PE" || !filepath.IsAbs(path) || !strings.Contains(strings.ToLower(path), `\hostedtoolcache\windows\python\`) || !strings.EqualFold(filepath.Base(path), "python.exe") {
		return r, fmt.Errorf("Python baseline is not an authorized hosted-runner interpreter")
	}
	code, out, err := runDirectProcess(ctx, path, []string{"-I", "--version"})
	r.Version = strings.TrimSpace(out)
	if err != nil || code != 0 || r.Version != "Python "+version {
		return r, fmt.Errorf("baseline Python version does not match the verified installer")
	}
	exe := filepath.Join(workRoot, "python-baseline-cleanup.exe")
	if err := os.Rename(filepath.Join(workRoot, "http-verified-installer.bin"), exe); err != nil {
		return r, err
	}
	logPath := filepath.Join(workRoot, "python-baseline-uninstall.log")
	args := []string{"/uninstall", "/quiet", "InstallAllUsers=1", "/log", logPath}
	r.Command = formatCommand(exe, args)
	r.ExitCode, r.Output, err = runDirectProcess(ctx, exe, args)
	r.Logs = vmPythonInstallerLogs(started, logPath)
	if err != nil || r.ExitCode != 0 {
		return r, fmt.Errorf("official Python baseline uninstall failed: exit=%d %v", r.ExitCode, err)
	}
	remaining, queryErr := registeredPython313Machine(ctx)
	_, fileErr := os.Stat(path)
	if queryErr != nil || remaining != "" || !os.IsNotExist(fileErr) {
		return r, fmt.Errorf("official Python baseline removal did not establish interpreter and registry absence")
	}
	r.Removed = true
	return r, nil
}
