package linuxpkg

import (
	"reflect"
	"testing"
)

func TestExactRemoveCommands(t *testing.T) {
	tests := []struct {
		provider string
		pkg      string
		remote   string
		want     Spec
	}{
		{"apt-get", "firefox", "", Spec{Name: "apt-get", Args: []string{"remove", "-y", "firefox"}, NeedsRoot: true}},
		{"dnf", "vlc", "", Spec{Name: "dnf", Args: []string{"remove", "-y", "vlc"}, NeedsRoot: true}},
		{"pacman", "krita", "", Spec{Name: "pacman", Args: []string{"-R", "--noconfirm", "krita"}, NeedsRoot: true}},
		{"zypper", "git", "", Spec{Name: "zypper", Args: []string{"--non-interactive", "remove", "git"}, NeedsRoot: true}},
		{"flatpak", "org.gimp.GIMP", "flathub", Spec{Name: "flatpak", Args: []string{"uninstall", "--user", "-y", "org.gimp.GIMP"}}},
	}
	for _, tc := range tests {
		got, err := Remove(tc.provider, tc.pkg, tc.remote)
		if err != nil {
			t.Fatalf("%s: %v", tc.provider, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: got %#v, want %#v", tc.provider, got, tc.want)
		}
	}
}

func TestDetectionIsExactAndIndependent(t *testing.T) {
	s, err := Detect("apt-get", "firefox", "")
	if err != nil {
		t.Fatal(err)
	}
	want := Spec{Name: "dpkg-query", Args: []string{"-W", "-f=${Status}", "firefox"}}
	if !reflect.DeepEqual(s, want) {
		t.Fatalf("detect spec = %#v, want %#v", s, want)
	}
	if DetectionSuccess("apt-get", []byte("deinstall ok config-files"), true) {
		t.Fatal("apt removed package must not be treated as installed")
	}
	if !DetectionSuccess("apt-get", []byte("install ok installed"), true) {
		t.Fatal("apt installed package should be detected")
	}
}

func TestRejectsUnsafeIdentifiers(t *testing.T) {
	for _, bad := range []string{"", "foo bar", "foo;rm", "$(evil)", "../pkg"} {
		if _, err := Remove("apt-get", bad, ""); err == nil {
			t.Fatalf("unsafe package id %q was accepted", bad)
		}
	}
	if _, err := Install("flatpak", "org.example.App", "flathub;evil", "Legújabb"); err == nil {
		t.Fatal("unsafe Flatpak remote was accepted")
	}
}
