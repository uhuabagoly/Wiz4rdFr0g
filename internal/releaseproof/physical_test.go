package releaseproof

import "testing"

func TestPortableInventoryRequiresExactIdentityAndSuccessfulQuery(t *testing.T) {
	zero := 0
	o := installedObservation{ID: "GodotEngine.GodotEngine", State: "present", ExitCode: 0x8a150014, InventoryExitCode: &zero, InventoryCommand: []string{"winget.exe", "list", "--accept-source-agreements", "--disable-interactivity"}, InventoryOutput: "Godot Engine  GodotEngine.GodotEngine  4.5  winget\n"}
	if !o.provesPresent() {
		t.Fatal("complete inventory proof rejected")
	}
	o.InventoryExitCode = nil
	if o.provesPresent() {
		t.Fatal("missing inventory exit accepted")
	}
	o.InventoryExitCode = &zero
	o.InventoryOutput = "Godot Engine GodotEngine.GodotEngine.Mono 4.5 winget"
	if o.provesPresent() {
		t.Fatal("different package accepted")
	}
	o.InventoryOutput = "Godot Engine GodotEngine.GodotEngine 4.5 winget"
	o.InventoryCommand = []string{"winget.exe", "search"}
	if o.provesPresent() {
		t.Fatal("repository search accepted as installed inventory")
	}
}
