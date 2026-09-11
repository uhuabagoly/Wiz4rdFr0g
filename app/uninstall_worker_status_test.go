package main

import (
	"errors"
	"os/exec"
	"testing"
)

func TestNormalizeWorkerExecutionTreatsNonZeroExitAsProtocolResult(t *testing.T) {
	code, err := normalizeWorkerExecution(uninstallCodeFailed, &exec.ExitError{})
	if err != nil || code != uninstallCodeFailed {
		t.Fatalf("got code=%d err=%v", code, err)
	}
}

func TestNormalizeWorkerExecutionPreservesTransportError(t *testing.T) {
	want := errors.New("CreateProcess failed")
	code, err := normalizeWorkerExecution(-1, want)
	if code != -1 || !errors.Is(err, want) {
		t.Fatalf("got code=%d err=%v", code, err)
	}
}
