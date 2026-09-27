//go:build windows

package main

import "testing"

func TestNSISRegisteredDirectoryTail(t *testing.T) {
	exe := `C:\Program Files\foobar2000\uninstall.exe`
	got, err := nsisRegisteredCommandLine(exe, `"C:\Program Files\foobar2000\uninstall.exe" _?=C:\Program Files\foobar2000`)
	want := `"C:\Program Files\foobar2000\uninstall.exe" /S _?=C:\Program Files\foobar2000`
	if err != nil || got != want {
		t.Fatalf("command = %q, error = %v", got, err)
	}
	if _, err := nsisRegisteredCommandLine(exe, `"`+exe+`" _?=relative`); err == nil {
		t.Fatal("relative tail accepted")
	}
}
