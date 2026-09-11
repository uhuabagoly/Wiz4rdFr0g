package main

import (
	"testing"
	"time"
)

func TestTransientPackageManagerFailureIsNarrow(t *testing.T) {
	if !isTransientPackageManagerFailure("HTTP 429 Too Many Requests") {
		t.Fatal("429 should be transient")
	}
	if !isTransientPackageManagerFailure("network is unreachable") {
		t.Fatal("network failure should be transient")
	}
	if isTransientPackageManagerFailure("package id mismatch") {
		t.Fatal("identity failure must not be retried as transient")
	}
}

func TestCampaignDurationBounds(t *testing.T) {
	t.Setenv("WF_TEST_TIMEOUT", "1")
	if got := campaignDurationFromEnv("WF_TEST_TIMEOUT", 30*time.Minute, 5*time.Minute, 60*time.Minute); got != 5*time.Minute {
		t.Fatalf("low bound=%s", got)
	}
	t.Setenv("WF_TEST_TIMEOUT", "999")
	if got := campaignDurationFromEnv("WF_TEST_TIMEOUT", 30*time.Minute, 5*time.Minute, 60*time.Minute); got != 60*time.Minute {
		t.Fatalf("high bound=%s", got)
	}
}

func TestCampaignRetryLimitIsCapped(t *testing.T) {
	t.Setenv("WF_TEST_RETRY", "99")
	if got := campaignRetryLimitFromEnv("WF_TEST_RETRY", 1); got != 2 {
		t.Fatalf("retry limit=%d", got)
	}
}

func TestUninstallRetryDecision(t *testing.T) {
	if !shouldRetryUninstall(uninstallCodeFailed, true, 1, 2) {
		t.Fatal("generic first-attempt failure should be retried")
	}
	if shouldRetryUninstall(uninstallCodeNeedElevation, true, 1, 2) {
		t.Fatal("privilege mismatch must not loop")
	}
	if shouldRetryUninstall(uninstallCodeFailed, true, 2, 2) {
		t.Fatal("retry limit must be enforced")
	}
}

func TestUninstallDiagnosisExplainsRepairableFailure(t *testing.T) {
	got := vmUninstallDiagnosis(uninstallCodeFailed, true, 2, 2)
	if got == "" || got == "exit status 1" {
		t.Fatalf("diagnosis is not actionable: %q", got)
	}
}
