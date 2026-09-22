package linuxpkg

import "testing"

func TestAPTCandidateUsesRepositoryPolicy(t *testing.T) {
	v, err := APTCandidate("audacity:\n  Installed: (none)\n  Candidate: 3.4.2\n  Version table:\n     3.7.3 100\n     3.4.2 500\n")
	if err != nil || v != "3.4.2" {
		t.Fatalf("%q %v", v, err)
	}
	if _, err = APTCandidate("Candidate: (none)"); err == nil {
		t.Fatal("missing candidate accepted")
	}
}

func TestDPKGRemovedRejectsBrokenAndInstalledStates(t *testing.T) {
	for _, s := range []string{"unknown ok not-installed", "deinstall ok config-files"} {
		if !DPKGRemoved(s, 0) {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"install ok installed", "deinstall reinstreq half-installed", "", "dpkg-query: error: failed to open package info file"} {
		if DPKGRemoved(s, 0) || DPKGRemoved(s, 1) || DPKGRemoved(s, 2) {
			t.Fatal(s)
		}
	}
	if !DPKGRemoved("dpkg-query: no packages found matching vlc", 1) {
		t.Fatal("exact absence rejected")
	}
}
