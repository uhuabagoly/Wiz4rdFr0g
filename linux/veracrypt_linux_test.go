//go:build linux

package main

import "testing"

func TestVeraCryptRequiresExactPublishedGUIPlatformAndSignature(t *testing.T) {
	base := "https://github.com/veracrypt/VeraCrypt/releases/download/VeraCrypt_1.26.29/"
	deb := base + "veracrypt-1.26.29-Ubuntu-24.04-amd64.deb"
	page := `<a href="` + deb + `">GUI</a><a href="` + deb + `.sig">Signature</a><a href="` + base + `veracrypt-console-1.26.29-Ubuntu-24.04-amd64.deb">Console</a>`
	got, version, err := veraCryptLinks(page, "Ubuntu-24.04-amd64")
	if err != nil || got != deb || version != "1.26.29" {
		t.Fatalf("wrong platform release: %s %s %v", got, version, err)
	}
	if _, _, err := veraCryptLinks(page, "Ubuntu-24.04-arm64"); err == nil {
		t.Fatal("accepted unpublished architecture")
	}
	if _, _, err := veraCryptLinks(`<a href="`+deb+`">GUI</a>`, "Ubuntu-24.04-amd64"); err == nil {
		t.Fatal("accepted missing publisher signature")
	}
}
