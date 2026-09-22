package linuxpkg

import "testing"

func TestInventoryErrorsCannotProveRemoval(t *testing.T) {
	for _, provider := range []string{"apt-get", "dnf", "zypper", "pacman", "flatpak"} {
		for _, code := range []int{1, 2, 127} {
			if _, err := InventoryContains(provider, "example", nil, code); err == nil {
				t.Fatalf("%s exit %d accepted", provider, code)
			}
		}
	}
}

func TestInventoryState(t *testing.T) {
	for _, tc := range []struct {
		provider, output string
		want             bool
	}{
		{"apt-get", "example\tinstall ok installed\n", true},
		{"apt-get", "example\tdeinstall ok config-files\n", false},
		{"apt-get", "example-helper\tinstall ok installed\n", false},
		{"dnf", "example\nexample-helper\n", true},
		{"pacman", "example-helper\n", false},
		{"flatpak", "example\n", true},
	} {
		got, err := InventoryContains(tc.provider, "example", []byte(tc.output), 0)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: got %v, %v", tc, got, err)
		}
	}
	if _, err := InventoryContains("apt-get", "example", []byte("database broken"), 0); err == nil {
		t.Fatal("malformed inventory accepted")
	}
	if _, err := InventoryContains("apt-get", "example", []byte("example\tinstall reinstreq half-installed\n"), 0); err == nil {
		t.Fatal("half-installed package accepted")
	}
	if DetectionSuccess("apt-get", []byte("not install ok installed"), true) {
		t.Fatal("substring treated as installed")
	}
}
