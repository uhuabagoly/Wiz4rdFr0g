package selfinstall

import "testing"

func TestWindowsPathComparison(t *testing.T) {
	good := `C:\Program Files\Wiz4rd Fr0g\Wiz4rdFr0g.exe`
	if !SameWindowsPath(good, `c:/program files/Wiz4rd Fr0g/Wiz4rdFr0g.exe`) {
		t.Fatal("equivalent Windows paths should match")
	}
	if SameWindowsPath(good, `D:\Tools\Wiz4rdFr0g.exe`) {
		t.Fatal("same filename in a foreign directory must not match")
	}
	if !PathWithinWindowsRoot(good, `C:\Program Files\Wiz4rd Fr0g`) {
		t.Fatal("installed executable should be within install root")
	}
	if PathWithinWindowsRoot(`C:\Program Files\Wiz4rd Fr0g Evil\Wiz4rdFr0g.exe`, `C:\Program Files\Wiz4rd Fr0g`) {
		t.Fatal("prefix sibling must not be treated as inside install root")
	}
}

func TestUninstallCommands(t *testing.T) {
	p := `C:\Program Files\Wiz4rd Fr0g\Uninstall.exe`
	if got, want := UninstallCommand(p, false), `"C:\Program Files\Wiz4rd Fr0g\Uninstall.exe" --uninstall`; got != want {
		t.Fatalf("interactive command = %q, want %q", got, want)
	}
	if got, want := UninstallCommand(p, true), `"C:\Program Files\Wiz4rd Fr0g\Uninstall.exe" --uninstall --quiet`; got != want {
		t.Fatalf("quiet command = %q, want %q", got, want)
	}
}
