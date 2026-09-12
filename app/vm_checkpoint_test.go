package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckpointReplacement(t *testing.T) {
	p := filepath.Join(t.TempDir(),"nested","result.json")
	for _, data := range []string{`{"phase":"INSTALL_PENDING"}`,`{"phase":"UNINSTALL_PENDING"}`} {
		if err := vmAtomicWrite(p,[]byte(data)); err != nil { t.Fatal(err) }
		b,err := os.ReadFile(p); if err != nil || string(b) != data { t.Fatalf("checkpoint not replaced: %s %v",b,err) }
	}
}
