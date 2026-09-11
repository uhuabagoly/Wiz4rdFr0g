package quality

import "testing"

func TestRootCauseClassifierKnownFailures(t *testing.T) {
	cases := []struct {
		name string
		f    ResultFacts
		want RootCause
	}{
		{"firefox-user-scope", ResultFacts{FinalStatus: "UNINSTALL_FAIL", FailureStage: "UNINSTALL", Failure: "The package installed for user scope cannot be uninstalled when running with administrator privileges."}, RootUserScopeElevated},
		{"edge-system", ResultFacts{FinalStatus: "SYSTEM_COMPONENT", SkipReason: "Windows-managed system component"}, RootSystemComponent},
		{"quoted-uninstall", ResultFacts{FinalStatus: "UNINSTALL_FAIL", FailureStage: "UNINSTALL", Failure: "Registry UninstallString quoted path parse failed"}, RootBrokenUninstallString},
		{"running", ResultFacts{FinalStatus: "UNINSTALL_FAIL", FailureStage: "UNINSTALL", Failure: "Please close the application because it is running"}, RootRunningProcess},
		{"reboot", ResultFacts{FinalStatus: "SKIPPED_REBOOT_REQUIRED", SkipReason: "installer reported reboot required"}, RootRebootRequired},
		{"detection", ResultFacts{FinalStatus: "DETECTION_FAIL", FailureStage: "DETECTION", Failure: "production detector failed"}, RootDetectionFalseNegative},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyRootCause(tc.f); got != tc.want {
				t.Fatalf("root cause=%s want=%s", got, tc.want)
			}
		})
	}
}

func TestCoverageNeverFabricatesVerifiedFull(t *testing.T) {
	if got := CoverageFor(ResultFacts{FinalStatus: "FULL_PASS", InstallVerified: true}); got == CoverageVerifiedFull {
		t.Fatal("FULL_PASS without uninstall verification became VERIFIED_FULL")
	}
	if got := CoverageFor(ResultFacts{FinalStatus: "FULL_PASS", InstallVerified: true, UninstallVerified: true}); got != CoverageVerifiedFull {
		t.Fatalf("verified physical result coverage=%s", got)
	}
}
