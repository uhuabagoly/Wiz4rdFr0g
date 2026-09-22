package main

import "testing"

func TestSplitRegisteredCommandQuoted(t *testing.T) {
	exe, args, err := splitRegisteredCommandRaw(`"C:\\Program Files\\Vendor App\\uninstall.exe" /S /norestart`)
	if err != nil {
		t.Fatal(err)
	}
	if exe != `C:\\Program Files\\Vendor App\\uninstall.exe` {
		t.Fatalf("exe = %q", exe)
	}
	if len(args) != 2 || args[0] != "/S" || args[1] != "/norestart" {
		t.Fatalf("args = %v", args)
	}
}

func TestSplitRegisteredCommandUnquotedPathWithSpaces(t *testing.T) {
	exe, args, err := splitRegisteredCommandRaw(`C:\\Program Files\\Vendor App\\uninstall.exe /remove /quiet`)
	if err != nil {
		t.Fatal(err)
	}
	if exe != `C:\\Program Files\\Vendor App\\uninstall.exe` {
		t.Fatalf("exe = %q", exe)
	}
	if len(args) != 2 || args[0] != "/remove" || args[1] != "/quiet" {
		t.Fatalf("args = %v", args)
	}
}

func TestMSIProductCode(t *testing.T) {
	guid := `{A1234567-B123-C123-D123-E123456789AB}`
	if got := extractMSIProductCode(`MsiExec.exe /I` + guid); got != guid {
		t.Fatalf("guid = %q", got)
	}
}

func TestWingetPortableRegisteredUninstaller(t *testing.T) {
	exe, args, err := splitRegisteredCommandRaw("winget uninstall --product-code Kubernetes.kind_Microsoft.Winget.Source_8wekyb3d8bbwe")
	if err != nil || exe != "winget.exe" || len(args) != 6 || args[2] != "Kubernetes.kind_Microsoft.Winget.Source_8wekyb3d8bbwe" {
		t.Fatalf("portable registration not preserved: %q %v %v", exe, args, err)
	}
	if _, _, err := splitRegisteredCommandRaw("winget uninstall --all"); err == nil {
		t.Fatal("unscoped extensionless command accepted")
	}
}
