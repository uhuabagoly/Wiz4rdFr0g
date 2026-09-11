package main

import "testing"

func TestUninstallWrongElevationFirefoxMessage(t *testing.T) {
	out := "The package installed for user scope cannot be uninstalled when running with administrator privileges."
	if !uninstallWrongElevation(out) {
		t.Fatal("user-scope admin mismatch was not detected")
	}
}

func TestUninstallNeedsElevation(t *testing.T) {
	for _, out := range []string{"Access is denied", "This operation requires administrator privileges", "Elevation required", "0x800702e4"} {
		if !uninstallNeedsElevation(out) {
			t.Fatalf("elevation requirement not detected: %s", out)
		}
	}
}

func TestUninstallExitSuccess(t *testing.T) {
	for _, code := range []int{0, 1641, 3010} {
		if !uninstallExitSuccess(code) {
			t.Fatalf("expected success for %d", code)
		}
	}
	for _, code := range []int{1, 1603, 1605} {
		if uninstallExitSuccess(code) {
			t.Fatalf("unexpected success for %d", code)
		}
	}
}

func TestUninstallRunningProcess(t *testing.T) {
	for _, out := range []string{"Please close the application before continuing", "Application is running", "Files are in use"} {
		if !uninstallRunningProcess(out) {
			t.Fatalf("running-process condition not detected: %s", out)
		}
	}
	if uninstallRunningProcess("package not found") {
		t.Fatal("unrelated error classified as running process")
	}
}

func TestClassifyUninstallAttempt(t *testing.T) {
	cases := []struct {
		a    uninstallAttempt
		want int
	}{
		{uninstallAttempt{Success: true}, uninstallCodeOK},
		{uninstallAttempt{Success: true, RebootRequired: true}, uninstallCodeRebootRequired},
		{uninstallAttempt{Output: "installed for user scope cannot be uninstalled when running with administrator privileges"}, uninstallCodeWrongElevation},
		{uninstallAttempt{Output: "Access is denied; elevation required"}, uninstallCodeNeedElevation},
		{uninstallAttempt{Output: "Please close the application because it is running"}, uninstallCodeRunningProcess},
		{uninstallAttempt{Output: "generic failure"}, uninstallCodeFailed},
	}
	for _, tc := range cases {
		if got := classifyUninstallAttempt(tc.a); got != tc.want {
			t.Fatalf("classify=%d want=%d for %+v", got, tc.want, tc.a)
		}
	}
}
