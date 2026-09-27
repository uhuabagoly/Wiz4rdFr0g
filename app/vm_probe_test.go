package main

import "testing"

func TestIndependentGodotInventoryIdentity(t *testing.T) {
	output := "Name             Id                       Version Source\r\n--------------------------------------------------------\r\nGodot Engine     GodotEngine.GodotEngine  4.7.2   winget\r\n"
	if got := independentWingetState("GodotEngine.GodotEngine", 0, output, false); got != "present" {
		t.Fatalf("actual installed inventory identity was missed: %s", got)
	}
	if got := independentWingetState("GodotEngine.Godot", 0, output, false); got != "unknown" {
		t.Fatalf("partial package identity accepted: %s", got)
	}
}

func TestIndependentWingetObservationFailsClosed(t *testing.T) {
	for _, c := range []struct {
		code   int
		output string
		failed bool
		want   string
	}{
		{0, "Name Vendor.App 1.0 winget\n", false, "present"},
		{0, "Name Vendor.App.Helper 1.0 winget\n", false, "unknown"},
		{0, "", false, "unknown"},
		{-1978335212, "No packages found", true, "absent"},
		{-1978335231, "Source failed", true, "unknown"},
		{-1, "Name Vendor.App 1.0 winget\n", true, "unknown"},
	} {
		if got := independentWingetState("Vendor.App", c.code, c.output, c.failed); got != c.want {
			t.Fatalf("code %d output %q: %s, want %s", c.code, c.output, got, c.want)
		}
	}
}
