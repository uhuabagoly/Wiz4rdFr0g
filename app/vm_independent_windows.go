//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type vmIndependentState struct {
	ID string `json:"id"`
	State string `json:"state"`
	Command []string `json:"command"`
	ExitCode int `json:"exit_code"`
	Output string `json:"output"`
	Error string `json:"error,omitempty"`
}

// This exact-ID observation deliberately does not use production matching/parsing.
// A source/query failure is UNKNOWN, never evidence that an application is absent.
func vmIndependentProbe(id, source string) vmIndependentState {
	a := []string{"list", "--id", id, "--exact", "--source", source, "--accept-source-agreements", "--disable-interactivity"}
	r := vmIndependentState{ID: id, State: "unknown", Command: append([]string{"winget.exe"}, a...)}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second); defer cancel()
	code, out, err := runDirectProcess(ctx, "winget.exe", a)
	r.ExitCode, r.Output = code, out
	if err != nil { r.Error = err.Error() }
	if ctx.Err() != nil { return r }
	r.State = independentWingetState(id, code, out, err != nil)
	return r
}

func vmActionsEnvironment(idx int) (json.RawMessage, error) {
	wantRun := "github-"+os.Getenv("GITHUB_RUN_ID")+"-"+os.Getenv("GITHUB_RUN_ATTEMPT")
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || os.Getenv("RUNNER_OS") != "Windows" || os.Getenv("RUNNER_ARCH") != "X64" || os.Getenv("GITHUB_RUN_ID") == "" || os.Getenv("GITHUB_RUN_ATTEMPT") == "" {
		return nil, fmt.Errorf("approved GitHub-hosted Windows x64 job required")
	}
	if os.Getenv("WIZ4RDFR0G_TEST_RUN_ID") != wantRun || os.Getenv("WIZ4RDFR0G_VM_ID") != fmt.Sprintf("%s-%d", wantRun, idx) { return nil, fmt.Errorf("Actions run/VM identity mismatch") }
	b, err := os.ReadFile(os.Getenv("WIZ4RDFR0G_ENVIRONMENT_EVIDENCE")); if err != nil { return nil, err }
	var e struct { Status string `json:"status"`; VMID string `json:"vm_id"`; Commit string `json:"git_commit"` }
	if err := json.Unmarshal(b, &e); err != nil { return nil, err }
	if e.Status != "READY" || e.VMID != os.Getenv("WIZ4RDFR0G_VM_ID") || e.Commit != buildGitCommit { return nil, fmt.Errorf("preflight missing or mismatched") }
	return json.RawMessage(b), nil
}
