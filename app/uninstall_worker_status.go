package main

import (
	"errors"
	"os/exec"
)

// normalizeWorkerExecution separates a worker's application exit status from
// failures to start/wait for the worker process. exec.Command returns an
// *exec.ExitError for every non-zero exit code; those codes are part of the
// uninstall protocol and must not be shown as "worker launch failed".
func normalizeWorkerExecution(code int, err error) (int, error) {
	if err == nil {
		return code, nil
	}
	var exitErr *exec.ExitError
	if code >= 0 && errors.As(err, &exitErr) {
		return code, nil
	}
	return code, err
}
