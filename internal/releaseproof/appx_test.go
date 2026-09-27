package releaseproof

import "testing"

func TestAppxProofRequiresPublisherAndManifestPayload(t *testing.T) {
	a := &AppxProof{Family: "ProtonPass_158qdr94jw63p", FullName: "ProtonPass_1.41.1.0_x64__158qdr94jw63p", InstallLocation: `C:\Program Files\WindowsApps\ProtonPass_1.41.1.0_x64__158qdr94jw63p`, Executables: []string{"ProtonPass.exe"}, Present: true, Removed: true}
	paths := []string{a.InstallLocation + `\ProtonPass.exe`}
	if !validAppxLifecycle("Proton.ProtonPass", a, paths) {
		t.Fatal("valid deployment rejected")
	}
	if validAppxLifecycle("Other.Package", a, paths) {
		t.Fatal("unrelated identity accepted")
	}
	if validAppxLifecycle("Proton.ProtonPass", a, []string{`C:\Other\installer.exe`}) {
		t.Fatal("unrelated file accepted")
	}
	a.Removed = false
	if validAppxLifecycle("Proton.ProtonPass", a, paths) {
		t.Fatal("remaining registration accepted")
	}
	a.Removed = true
	a.FullName = "ProtonPass_1.0_x64__different"
	if validAppxLifecycle("Proton.ProtonPass", a, paths) {
		t.Fatal("different publisher accepted")
	}
}
