package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSigningConfigStates(t *testing.T) {
	t.Setenv(signingToolEnv, "")
	cfg, err := loadSigningConfig()
	if err != nil || cfg.Tool != "" {
		t.Fatalf("disabled signing config = %#v, %v", cfg, err)
	}

	t.Setenv(signingToolEnv, "signtool.exe")
	t.Setenv(signingArgsEnv, "")
	if _, err := loadSigningConfig(); err == nil {
		t.Fatal("configured signing tool without arguments must fail closed")
	}

	t.Setenv(signingArgsEnv, `["sign","/fd","SHA256"]`)
	t.Setenv(signingVerifyArgsEnv, `["verify","/pa"]`)
	cfg, err = loadSigningConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tool != "signtool.exe" || len(cfg.SignArgs) != 3 || len(cfg.VerifyArgs) != 2 {
		t.Fatalf("unexpected parsed signing config: %#v", cfg)
	}
}

func TestWriteSHA256SumsCoversEveryPayload(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "Wiz4rdFr0g"),
		filepath.Join(dir, "Wiz4rdFr0g_icon.png"),
		filepath.Join(dir, "install.sh"),
		filepath.Join(dir, "uninstall.sh"),
	}
	for i, p := range files {
		if err := os.WriteFile(p, []byte(strings.Repeat(string(rune('a'+i)), i+1)), 0644); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(dir, "SHA256SUMS")
	if err := writeSHA256Sums(out, files); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, p := range files {
		base := filepath.Base(p)
		if !strings.Contains(text, "  "+base+"\n") {
			t.Fatalf("checksum manifest does not cover %s: %s", base, text)
		}
	}
}
